# Crypto Currency Portfolio Tracker

A lightweight, terminal-based cryptocurrency portfolio management application built with Go. Track your investments, monitor real-time prices, and analyze your portfolio performance—all from the command line.

## 🚀 Features

- **Real-Time Price Tracking**: Fetch live cryptocurrency prices from market APIs
- **Portfolio Management**: Add, update, and remove cryptocurrency holdings
- **Profit/Loss Calculation**: Automatically calculate gains and losses on your investments
- **User Authentication**: Secure login system with encrypted credentials
- **Persistent Storage**: MongoDB integration for reliable data persistence
- **Export Functionality**: Export portfolio data to CSV/JSON formats
- **Terminal UI**: Clean, menu-driven interface for easy navigation
- **Concurrent Operations**: Efficient parallel processing for price updates

## 📋 Table of Contents

- [Problem Domain](#problem-domain)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [Project Structure](#project-structure)
- [Architecture](#architecture)
- [API Integration](#api-integration)
- [Contributing](#contributing)
- [License](#license)

## 🎯 Problem Domain

Most cryptocurrency portfolio tracking tools are either:
- UI-heavy web applications that consume significant resources
- Lack proper data persistence across sessions
- Require constant internet connectivity for basic operations

This project addresses these limitations by providing a **lightweight, secure, terminal-based solution** that:
- Runs efficiently with minimal system resources
- Offers persistent storage using MongoDB
- Supports real-time data processing without a graphical interface
- Focuses on core functionality without unnecessary bloat

## ✅ Prerequisites

Before you begin, ensure you have the following installed:

- **Go** (version 1.19 or higher)
- **MongoDB** (version 4.4 or higher)
- **Git** (for cloning the repository)

Optional:
- **Make** (for using Makefile commands)

## 📥 Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/crypto-portfolio-tracker.git
cd crypto-portfolio-tracker
```

2. Install Go dependencies:
```bash
go mod download
```

3. Set up MongoDB:
```bash
# Start MongoDB service (Linux/macOS)
sudo systemctl start mongod

# Or using Docker
docker run -d -p 27017:27017 --name mongodb mongo:latest
```

4. Build the application:
```bash
go build -o crypto-tracker ./cmd/main.go
```

## ⚙️ Configuration

Create a `config.yaml` file in the root directory:

```yaml
database:
  host: "localhost"
  port: 27017
  name: "crypto_portfolio"
  username: ""  # Optional
  password: ""  # Optional

api:
  provider: "coingecko"  # or "coinmarketcap"
  key: ""  # Required for some providers
  rate_limit: 50  # Requests per minute

security:
  jwt_secret: "your-secret-key-here"
  session_timeout: 3600  # Seconds

logging:
  level: "info"  # debug, info, warn, error
  file: "app.log"
```

Alternatively, use environment variables:
```bash
export DB_HOST=localhost
export DB_PORT=27017
export DB_NAME=crypto_portfolio
export API_PROVIDER=coingecko
export JWT_SECRET=your-secret-key
```

## 🎮 Usage

### Starting the CLI Application

```bash
./crypto-tracker
```

### Running the Go API Server (for React frontend)

```bash
go run main.go api
```

The API server will listen on port 8080 by default.

### Running the React Frontend

From the project root:

```bash
cd frontend
npm install
npm run dev
```

Then open the URL shown in the terminal (usually http://localhost:5173).

### Main Menu Options

```

=== Currently in development ===

```

<!-- ### Example Workflow

**1. Register/Login**
```bash
# First-time users will be prompted to create an account
Username: john_doe
Password: ********
```

**2. Add a Cryptocurrency**
```bash
Select option: 2
Enter coin symbol (e.g., BTC): BTC
Enter amount: 0.5
Enter purchase price (USD): 45000
```

**3. View Portfolio**
```bash
Select option: 1

Symbol  | Amount    | Avg Price  | Current Price | Value     | P/L      | P/L %
--------|-----------|------------|---------------|-----------|----------|-------
BTC     | 0.50000   | $45,000.00 | $47,500.00    | $23,750.00| +$1,250.00| +5.56%
ETH     | 5.00000   | $3,200.00  | $3,350.00     | $16,750.00| +$750.00 | +4.69%
--------|-----------|------------|---------------|-----------|----------|-------
Total   |           |            |               | $40,500.00| +$2,000.00| +5.19%
```

**4. Export Portfolio**
```bash
Select option: 7
Export format (csv/json): csv
File saved: portfolio_2025-02-13.csv
```

## 📁 Project Structure

```
crypto-portfolio-tracker/
├── cmd/
│   └── main.go                 # Application entry point
├── internal/
│   ├── auth/                   # Authentication logic
│   │   ├── jwt.go
│   │   └── user.go
│   ├── database/               # Database operations
│   │   ├── mongodb.go
│   │   └── repository.go
│   ├── models/                 # Data models
│   │   ├── coin.go
│   │   ├── portfolio.go
│   │   └── user.go
│   ├── api/                    # External API integration
│   │   ├── coingecko.go
│   │   └── price_fetcher.go
│   ├── calculator/             # P/L calculations
│   │   └── profit_loss.go
│   └── ui/                     # Terminal UI components
│       ├── menu.go
│       └── display.go
├── pkg/
│   ├── config/                 # Configuration management
│   │   └── config.go
│   └── utils/                  # Utility functions
│       ├── validation.go
│       └── export.go
├── tests/                      # Unit and integration tests
│   ├── auth_test.go
│   ├── calculator_test.go
│   └── api_test.go
├── scripts/                    # Helper scripts
│   └── setup_db.sh
├── config.yaml                 # Configuration file
├── go.mod                      # Go module dependencies
├── go.sum
├── Makefile                    # Build automation
├── LICENSE
└── README.md
```

## 🏗️ Architecture

The application follows **Clean Architecture** principles with clear separation of concerns:

### Layers

1. **Presentation Layer** (`internal/ui`): Terminal interface and user interactions
2. **Business Logic Layer** (`internal/calculator`, `internal/auth`): Core business rules
3. **Data Access Layer** (`internal/database`): Database operations and repositories
4. **External Services** (`internal/api`): Third-party API integrations

### Key Design Patterns

- **Repository Pattern**: Abstracts database operations
- **Dependency Injection**: Promotes testability and modularity
- **Factory Pattern**: Creates API client instances
- **Observer Pattern**: Real-time price updates

### Concurrency Model

The application leverages Go's concurrency features:

```go
// Example: Concurrent price fetching
func (p *Portfolio) RefreshPrices() error {
    var wg sync.WaitGroup
    results := make(chan PriceUpdate, len(p.Coins))
    
    for _, coin := range p.Coins {
        wg.Add(1)
        go func(c Coin) {
            defer wg.Done()
            price, err := p.api.FetchPrice(c.Symbol)
            results <- PriceUpdate{Symbol: c.Symbol, Price: price, Err: err}
        }(coin)
    }
    
    go func() {
        wg.Wait()
        close(results)
    }()
    
    // Process results...
}
```

## 🔌 API Integration

### Supported Providers

- **CoinGecko** (Default, no API key required)
- **CoinMarketCap** (Requires API key)
- **Binance** (Real-time data)

### Rate Limiting

The application implements intelligent rate limiting to respect API quotas:

```go
type RateLimiter struct {
    requests chan struct{}
    interval time.Duration
}

func (r *RateLimiter) Wait() {
    <-r.requests
    time.AfterFunc(r.interval, func() {
        r.requests <- struct{}{}
    })
}
```

### Error Handling

Robust error handling with automatic retries:
- Exponential backoff for failed requests
- Circuit breaker pattern for API failures
- Graceful degradation when APIs are unavailable

## 🧪 Testing

Run the test suite:

```bash
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./internal/calculator -v

# Run integration tests (requires MongoDB)
go test ./tests/integration -tags=integration
```

### Test Coverage Goals

- Unit Tests: >80% coverage
- Integration Tests: Critical paths
- End-to-End Tests: Main user workflows

## 🤝 Contributing

We welcome contributions! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Standards

- Follow Go conventions and style guidelines
- Write meaningful commit messages
- Add tests for new features
- Update documentation as needed
- Run `go fmt` and `go vet` before committing

### Development Workflow

```bash
# Install development tools
make install-tools

# Run linter
make lint

# Format code
make fmt

# Run tests
make test

# Build application
make build
```

## 📊 Performance Considerations

- **Memory Usage**: Optimized for <50MB RAM usage
- **Database Queries**: Indexed fields for fast lookups
- **Concurrent Operations**: Worker pool pattern for price updates
- **Caching**: In-memory cache for frequently accessed data

## 🔒 Security

- Passwords hashed using bcrypt (cost factor: 12)
- JWT tokens for session management
- MongoDB credentials stored securely
- Input validation and sanitization
- Rate limiting to prevent abuse

## 🗺️ Roadmap

- [ ] Support for more cryptocurrency exchanges
- [ ] Historical price charts in terminal
- [ ] Portfolio rebalancing suggestions
- [ ] Tax reporting features
- [ ] Multi-currency support (EUR, GBP, etc.)
- [ ] Portfolio diversification analysis
- [ ] Alert system for price movements
- [ ] REST API for external integrations

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👨‍💻 Author

**Your Name**
- GitHub: [@yourusername](https://github.com/yourusername)
- Email: your.email@example.com

## 🙏 Acknowledgments

- [CoinGecko API](https://www.coingecko.com/en/api) for cryptocurrency data
- [MongoDB](https://www.mongodb.com/) for database solutions
- [Cobra](https://github.com/spf13/cobra) for CLI framework
- The Go community for excellent libraries and tools

---

**Disclaimer**: This software is for educational purposes. Cryptocurrency investments carry risk. Always do your own research before making investment decisions. -->