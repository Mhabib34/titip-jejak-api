# Titip Jejak — API (Backend)

> **EN** | REST API backend for the Titip Jejak missing persons platform.  
> **ID** | Backend REST API untuk platform pencarian orang hilang Titip Jejak.

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL%20v3-orange.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/Mhabib34/titip-jejak-api/blob/main/CONTRIBUTING.md)

---

## 🇬🇧 English

### Overview

This is the backend API for [Titip Jejak](https://github.com/Mhabib34/titip-jejak-web), handling:

- Authentication (via Supabase Auth)
- Missing persons report CRUD
- Report matching logic
- Volunteer management
- Statistics endpoints

### Repositories

| Repo                                                             | Description                  |
| ---------------------------------------------------------------- | ---------------------------- |
| [`titip-jejak-web`](https://github.com/Mhabib34/titip-jejak-web) | Next.js frontend             |
| [`titip-jejak-api`](https://github.com/Mhabib34/titip-jejak-api) | This repo — REST API backend |

### Tech Stack

- **Language**: Go (Golang) >= 1.21
- **Database/Auth**: Supabase (PostgreSQL + Auth)

### Getting Started

#### Prerequisites

- Go >= 1.21 ([download](https://go.dev/dl/))
- A Supabase project ([create one free](https://supabase.com))

#### Installation

```bash
# 1. Clone the repo
git clone https://github.com/Mhabib34/titip-jejak-api.git
cd titip-jejak-api

# 2. Install dependencies
go mod tidy

# 3. Set up environment variables
cp .env.test .env
# Fill in your Supabase credentials in .env

# 4. Run the development server
go run ./cmd/api/main.go
```

#### Build for Production

```bash
go build -o titip-jejak-api ./cmd/api/main.go
./titip-jejak-api
```

#### Environment Variables

| Variable                | Description                            |
| ----------------------- | -------------------------------------- |
| `DATABASE_URL`          | Your Supabase project URL              |
| `PORT`                  | Port to run the API on (default: 8080) |
| `JWT_ACCESS_SECRET`     | JWT ACCESS SECRET                      |
| `JWT_REFRESH_SECRET`    | JWT REFRESH SECRET                     |
| `CLOUDINARY_API_KEY`    | Cloudinary api key                     |
| `CLOUDINARY_CLOUD_NAME` | Cloudinary cloud key                   |
| `CLOUDINARY_API_SECRET` | Cloudinary api secret key              |

### Contributing

Please read [CONTRIBUTING.md](./CONTRIBUTING.md) before submitting a pull request.

### License

Licensed under **GNU Affero General Public License v3.0**.  
See [LICENSE](./LICENSE) for details.

---

## 🇮🇩 Indonesia

### Gambaran Umum

Ini adalah backend API untuk [Titip Jejak](https://github.com/Mhabib34/titip-jejak-web), menangani autentikasi, laporan orang hilang, logika pencocokan antar laporan, manajemen relawan, dan endpoint statistik.

### Cara Menjalankan Secara Lokal

```bash
# 1. Clone repo
git clone https://github.com/Mhabib34/titip-jejak-api.git
cd titip-jejak-api

# 2. Install dependensi
go mod tidy

# 3. Salin dan isi environment variable
cp .env.test .env

# 4. Jalankan server
go run ./cmd/api/main.go
```

#### Build Production

```bash
go build -o titip-jejak-api ./cmd/api/main.go
./titip-jejak-api
```

### Lisensi

Menggunakan lisensi **GNU Affero General Public License v3.0**.

---

## Contact / Kontak

**Muhammad Habib**  
📧 [mhabib34official@gmail.com](mailto:mhabib34official@gmail.com)  
🐙 GitHub: [@Mhabib34](https://github.com/Mhabib34)
