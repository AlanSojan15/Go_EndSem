package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"crypto-portfolio-tracker/alert"
	"crypto-portfolio-tracker/api"
	"crypto-portfolio-tracker/auth"
	"crypto-portfolio-tracker/models"
	"crypto-portfolio-tracker/portfolio"
)

type Server struct {
	api      api.CryptoApi
	sessions map[string]string // token -> email
	mu       sync.RWMutex
}

type jsonResponse struct {
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

type addHoldingRequest struct {
	CoinID    string  `json:"coin_id"`
	CoinName  string  `json:"coin_name"`
	Quantity  float64 `json:"quantity"`
	PriceType string  `json:"price_type"` // p|t
	BuyPrice  float64 `json:"buy_price"`
}

type addAlertRequest struct {
	CoinID    string  `json:"coin_id"`
	AlertType string  `json:"alert_type"` // buy|sell
	Threshold float64 `json:"threshold"`
}

type importHolding struct {
	CoinID    string  `json:"coin_id"`
	CoinName  string  `json:"coin_name"`
	Quantity  float64 `json:"quantity"`
	BuyPrice  float64 `json:"buy_price"`
	PriceType string  `json:"price_type"`
}

type importRequest struct {
	Holdings []importHolding `json:"holdings"`
}

type holdingView struct {
	CoinID        string  `json:"coin_id"`
	CoinName      string  `json:"coin_name"`
	Quantity      float64 `json:"quantity"`
	BuyPrice      float64 `json:"buy_price"`
	CurrentPrice  float64 `json:"current_price"`
	CurrentValue  float64 `json:"current_value"`
	ProfitLoss    float64 `json:"profit_loss"`
	ProfitLossPct float64 `json:"profit_loss_pct"`
}

type portfolioResponse struct {
	Holdings   []holdingView `json:"holdings"`
	TotalValue float64       `json:"total_value"`
}

func Start(apiClient api.CryptoApi, port string) error {
	s := &Server{api: apiClient, sessions: make(map[string]string)}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/ping", s.handlePing)
	mux.HandleFunc("/api/signup", s.handleSignup)
	mux.HandleFunc("/api/login", s.handleLogin)
	mux.HandleFunc("/api/portfolio", s.authMiddleware(s.handlePortfolio))
	mux.HandleFunc("/api/portfolio/export", s.authMiddleware(s.handleExportPortfolio))
	mux.HandleFunc("/api/portfolio/import", s.authMiddleware(s.handleImportPortfolio))
	mux.HandleFunc("/api/portfolio/holdings", s.authMiddleware(s.handleAddHolding))
	mux.HandleFunc("/api/alerts", s.authMiddleware(s.handleAlerts))
	mux.HandleFunc("/api/alerts/check", s.authMiddleware(s.handleCheckAlerts))
	mux.HandleFunc("/api/alerts/delete", s.authMiddleware(s.handleDeleteAlert))

	return http.ListenAndServe(":"+port, s.cors(mux))
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, jsonResponse{Success: true, Data: "ok"})
}

func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSON(w, http.StatusMethodNotAllowed, jsonResponse{Success: false, Error: "method not allowed"})
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, jsonResponse{Success: false, Error: "invalid payload"})
		return
	}

	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
		s.writeJSON(w, http.StatusBadRequest, jsonResponse{Success: false, Error: "email and password are required"})
		return
	}

	if err := auth.SignupNoOTP(req.Email, req.Password); err != nil {
		s.writeJSON(w, http.StatusBadRequest, jsonResponse{Success: false, Error: err.Error()})
		return
	}

	s.writeJSON(w, http.StatusOK, jsonResponse{Success: true})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSON(w, http.StatusMethodNotAllowed, jsonResponse{Success: false, Error: "method not allowed"})
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, jsonResponse{Success: false, Error: "invalid payload"})
		return
	}

	if !auth.Login(req.Email, req.Password) {
		s.writeJSON(w, http.StatusUnauthorized, jsonResponse{Success: false, Error: "invalid credentials"})
		return
	}

	token, err := s.newToken(req.Email)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, jsonResponse{Success: false, Error: "could not create session"})
		return
	}

	s.writeJSON(w, http.StatusOK, jsonResponse{Success: true, Data: tokenResponse{Token: token}})
}

func (s *Server) handlePortfolio(w http.ResponseWriter, r *http.Request, email string) {
	if r.Method != http.MethodGet {
		s.writeJSON(w, http.StatusMethodNotAllowed, jsonResponse{Success: false, Error: "method not allowed"})
		return
	}

	p, err := portfolio.GetPortfolio(email)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, jsonResponse{Success: false, Error: err.Error()})
		return
	}

	if len(p.Holdings) == 0 {
		s.writeJSON(w, http.StatusOK, jsonResponse{Success: true, Data: portfolioResponse{Holdings: []holdingView{}, TotalValue: 0}})
		return
	}

	coinIDs := make([]string, len(p.Holdings))
	for i, h := range p.Holdings {
		coinIDs[i] = h.CoinID
	}

	prices, err := s.api.FetchMultiplePrices(coinIDs...)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, jsonResponse{Success: false, Error: err.Error()})
		return
	}

	holdings := make([]holdingView, 0, len(p.Holdings))
	var total float64
	for _, h := range p.Holdings {
		price := prices[h.CoinID]
		current := price * h.Quantity
		invested := h.BuyPrice * h.Quantity
		pl := current - invested
		plPct := 0.0
		if invested != 0 {
			plPct = (pl / invested) * 100
		}
		holdings = append(holdings, holdingView{
			CoinID:        h.CoinID,
			CoinName:      h.CoinName,
			Quantity:      h.Quantity,
			BuyPrice:      h.BuyPrice,
			CurrentPrice:  price,
			CurrentValue:  current,
			ProfitLoss:    pl,
			ProfitLossPct: plPct,
		})
		total += current
	}

	s.writeJSON(w, http.StatusOK, jsonResponse{Success: true, Data: portfolioResponse{Holdings: holdings, TotalValue: total}})
}

func (s *Server) handleAddHolding(w http.ResponseWriter, r *http.Request, email string) {
	if r.Method != http.MethodPost {
		s.writeJSON(w, http.StatusMethodNotAllowed, jsonResponse{Success: false, Error: "method not allowed"})
		return
	}

	var req addHoldingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, jsonResponse{Success: false, Error: "invalid payload"})
		return
	}

	if strings.TrimSpace(req.CoinID) == "" || strings.TrimSpace(req.CoinName) == "" || req.Quantity <= 0 || req.BuyPrice <= 0 {
		s.writeJSON(w, http.StatusBadRequest, jsonResponse{Success: false, Error: "invalid input"})
		return
	}

	buyPrice := req.BuyPrice
	if strings.ToLower(strings.TrimSpace(req.PriceType)) == "t" {
		buyPrice = buyPrice / req.Quantity
	}

	holding := models.Holding{
		CoinID:   req.CoinID,
		CoinName: req.CoinName,
		Quantity: req.Quantity,
		BuyPrice: buyPrice,
	}

	if err := portfolio.AddMultipleHoldings(email, holding); err != nil {
		s.writeJSON(w, http.StatusBadRequest, jsonResponse{Success: false, Error: err.Error()})
		return
	}

	s.writeJSON(w, http.StatusOK, jsonResponse{Success: true})
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request, email string) {
	if r.Method == http.MethodGet {
		alerts, err := alert.GetAlerts(email)
		if err != nil {
			s.writeJSON(w, http.StatusInternalServerError, jsonResponse{Success: false, Error: err.Error()})
			return
		}
		s.writeJSON(w, http.StatusOK, jsonResponse{Success: true, Data: alerts})
		return
	}

	if r.Method == http.MethodPost {
		var req addAlertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeJSON(w, http.StatusBadRequest, jsonResponse{Success: false, Error: "invalid payload"})
			return
		}

		if strings.TrimSpace(req.CoinID) == "" || req.Threshold <= 0 {
			s.writeJSON(w, http.StatusBadRequest, jsonResponse{Success: false, Error: "invalid input"})
			return
		}

		alertType := models.AlertTypeSell
		if strings.ToLower(strings.TrimSpace(req.AlertType)) == "buy" {
			alertType = models.AlertTypeBuy
		}

		coinName := req.CoinID
		p, _ := portfolio.GetPortfolio(email)
		for _, h := range p.Holdings {
			if h.CoinID == req.CoinID {
				coinName = h.CoinName
				break
			}
		}

		if err := alert.CreateAlert(email, req.CoinID, coinName, alertType, req.Threshold, s.api); err != nil {
			s.writeJSON(w, http.StatusBadRequest, jsonResponse{Success: false, Error: err.Error()})
			return
		}

		s.writeJSON(w, http.StatusOK, jsonResponse{Success: true})
		return
	}

	s.writeJSON(w, http.StatusMethodNotAllowed, jsonResponse{Success: false, Error: "method not allowed"})
}

func (s *Server) handleCheckAlerts(w http.ResponseWriter, r *http.Request, email string) {
	if r.Method != http.MethodPost {
		s.writeJSON(w, http.StatusMethodNotAllowed, jsonResponse{Success: false, Error: "method not allowed"})
		return
	}

	if err := alert.CheckAndTriggerAlerts(email, s.api); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, jsonResponse{Success: false, Error: err.Error()})
		return
	}

	s.writeJSON(w, http.StatusOK, jsonResponse{Success: true})
}

func (s *Server) handleDeleteAlert(w http.ResponseWriter, r *http.Request, email string) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		s.writeJSON(w, http.StatusMethodNotAllowed, jsonResponse{Success: false, Error: "method not allowed"})
		return
	}

	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.ID) == "" {
		s.writeJSON(w, http.StatusBadRequest, jsonResponse{Success: false, Error: "alert id required"})
		return
	}

	if err := alert.DeleteAlert(email, body.ID); err != nil {
		s.writeJSON(w, http.StatusBadRequest, jsonResponse{Success: false, Error: err.Error()})
		return
	}

	s.writeJSON(w, http.StatusOK, jsonResponse{Success: true})
}

func (s *Server) handleExportPortfolio(w http.ResponseWriter, r *http.Request, email string) {
	if r.Method != http.MethodGet {
		s.writeJSON(w, http.StatusMethodNotAllowed, jsonResponse{Success: false, Error: "method not allowed"})
		return
	}

	p, err := portfolio.GetPortfolio(email)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, jsonResponse{Success: false, Error: err.Error()})
		return
	}

	type exportHolding struct {
		CoinID   string  `json:"coin_id"`
		CoinName string  `json:"coin_name"`
		Quantity float64 `json:"quantity"`
		BuyPrice float64 `json:"buy_price"`
	}
	type exportPayload struct {
		UserEmail string          `json:"user_email"`
		ExportedAt string         `json:"exported_at"`
		Holdings  []exportHolding `json:"holdings"`
	}

	holdingsOut := make([]exportHolding, len(p.Holdings))
	for i, h := range p.Holdings {
		holdingsOut[i] = exportHolding{
			CoinID:   h.CoinID,
			CoinName: h.CoinName,
			Quantity: h.Quantity,
			BuyPrice: h.BuyPrice,
		}
	}

	payload := exportPayload{
		UserEmail:  email,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Holdings:   holdingsOut,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="portfolio.json"`)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) handleImportPortfolio(w http.ResponseWriter, r *http.Request, email string) {
	if r.Method != http.MethodPost {
		s.writeJSON(w, http.StatusMethodNotAllowed, jsonResponse{Success: false, Error: "method not allowed"})
		return
	}

	var req importRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, jsonResponse{Success: false, Error: "invalid JSON payload"})
		return
	}

	if len(req.Holdings) == 0 {
		s.writeJSON(w, http.StatusBadRequest, jsonResponse{Success: false, Error: "no holdings provided"})
		return
	}

	holdings := make([]models.Holding, 0, len(req.Holdings))
	for _, ih := range req.Holdings {
		if strings.TrimSpace(ih.CoinID) == "" || ih.Quantity <= 0 || ih.BuyPrice <= 0 {
			continue
		}
		buyPrice := ih.BuyPrice
		if strings.ToLower(strings.TrimSpace(ih.PriceType)) == "t" {
			buyPrice = buyPrice / ih.Quantity
		}
		coinName := ih.CoinName
		if coinName == "" {
			coinName = ih.CoinID
		}
		holdings = append(holdings, models.Holding{
			CoinID:   strings.ToLower(strings.TrimSpace(ih.CoinID)),
			CoinName: coinName,
			Quantity: ih.Quantity,
			BuyPrice: buyPrice,
		})
	}

	if len(holdings) == 0 {
		s.writeJSON(w, http.StatusBadRequest, jsonResponse{Success: false, Error: "no valid holdings in payload"})
		return
	}

	if err := portfolio.AddMultipleHoldings(email, holdings...); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, jsonResponse{Success: false, Error: err.Error()})
		return
	}

	s.writeJSON(w, http.StatusOK, jsonResponse{Success: true, Data: map[string]int{"imported": len(holdings)}})
}

func (s *Server) newToken(email string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b)

	s.mu.Lock()
	s.sessions[token] = email
	s.mu.Unlock()

	return token, nil
}

func (s *Server) getEmailFromToken(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	email, ok := s.sessions[token]
	return email, ok
}

func (s *Server) authMiddleware(next func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if auth == "" {
			s.writeJSON(w, http.StatusUnauthorized, jsonResponse{Success: false, Error: "authorization required"})
			return
		}

		parts := strings.Fields(auth)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			s.writeJSON(w, http.StatusUnauthorized, jsonResponse{Success: false, Error: "invalid authorization header"})
			return
		}

		email, ok := s.getEmailFromToken(parts[1])
		if !ok {
			s.writeJSON(w, http.StatusUnauthorized, jsonResponse{Success: false, Error: "invalid token"})
			return
		}

		next(w, r, email)
	}
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, payload jsonResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
