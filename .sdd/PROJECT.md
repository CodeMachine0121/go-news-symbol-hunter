# Project Overview

**Project:** go-symbol-news-hunter
**Bounded Context:** 標的資訊面分析（Symbol News Analysis）——依標的與市場類別爬取新聞，交由 AI 分析並產出標準化評等
**Last Updated:** 2026-10-06
**Status:** Draft

---

## 1. Vision & Mission

- **Problem Statement:** 股票 / 加密貨幣的資訊面散落在各家新聞平台，逐一閱讀耗時且容易只看到單方面說法。本專案依「標的 + 市場類別」自動從對應的新聞來源取回新聞，交由 AI 透過工具分析，產出**標準化、可追溯來源**的評等與理由。
- **Target Users:** 對外提供 API 的使用者。比照業界 API 服務，使用者自行建立 API key，**經 administrator 啟用後**才能呼叫分析。
- **Success Metrics:** **評等可回測驗證**——每次分析記錄分析時價格，日後能回頭比對評等方向的命中率，且命中率需優於隨機。
- **Out of Scope:**
  - 不代操、不下單：純資訊 / 決策輔助。
  - 不做技術面 / 籌碼面分析：只看資訊面（新聞），不碰 K 線、指標、法人買賣。

---

## 2. Core Tech Stack

只記錄已拍板的項目；標 `TBD` 的尚未決定，不得自行假設。

| Layer | Technology | Version / Notes |
| :--- | :--- | :--- |
| Frontend | 無 | 純後端 REST API |
| Backend | Go 1.26 · Gin | REST API |
| Database | PostgreSQL · GORM（`gorm.io/driver/postgres`） | 不使用 SQLite；Code First，啟動時 `AutoMigrate`；連線字串 `DATABASE_URL` |
| Infrastructure | 本機 `go run` | 部署方式 TBD |
| Key Libraries | `shopspring/decimal`（金額欄位）、`joho/godotenv`（讀 `.env`） | 測試：`testing` + `stretchr/testify`；mock 由 `mockery` 依介面產生於 `internal/domain/interface/mocks/` |
| External Services | Anthropic（暫定，AI 分析 + tool use）、各市場新聞來源 | 新聞來源清單於功能切片拍板 |

---

## 3. Architecture Principles

詳細規範見 `.claude/rules/`，此處只列要點。

- **Style:** 單體（Modular Monolith），單一 Go 服務。
- **Key Patterns:**
  - Clean / Onion Architecture：Controller → Application → Domain ← Infrastructure，依賴一律指向 Domain。
  - Entity 為乾淨的 Data Model，業務行為放 Domain Model；Domain Service 是 Application 的唯一呼叫入口，回傳 DTO。
  - 外部資源一律 Proxy，介面以能力命名（如新聞來源介面不綁供應商），實作帶供應商前綴。
  - Repository：一個 entity 一個 repository，Code First。
- **Folder / Layer Structure:** 原始碼在 `internal/`（domain / application / controller / infrastructure / job），入口與組裝根在 `cmd/`。
- **Data Flow:**
  1. 使用者帶 API key 呼叫分析（標的、市場類別）。
  2. 驗證 API key 存在且已啟用。
  3. 同標的在重用時間窗內已有分析結果 → 直接回傳，不重跑 AI。
  4. 否則建立分析事件 → AI 以「爬取新聞」工具取得新聞（程式依市場類別選擇新聞來源）→ AI 回傳結構化 JSON → Domain Model 正規化 → 落地分析結果。

---

## 4. Development Conventions

- **Naming:** 全名不縮寫；角色固定後綴（`Service` / `Application` / `Controller` / `Repository` / `Proxy`）；model 後綴 entity（無）/ `Domain` / `Dto` / `Vo` / `Request`；業務詞彙以 `.sdd/UL-MAP.md` 為準。
- **Branching Strategy:** TBD
- **Testing Requirements:** SDD + TDD；只測業務行為；只用 mocking 套件 mock 介面、禁手寫 Fake；Application 測試注入真實 Domain Service（測試力度放大）；測試放各層 `tests/`、外部 `<pkg>_test` package。
- **Code Review Rules:** 無 CI / pre-commit hook；commit 前手動確認 `go build ./...`、`go vet ./...`、`go test ./...` 全過。

---

## 5. Non-Negotiables & Constraints

- **Performance SLAs:** TBD。單次分析含多來源爬取與 AI 多輪工具呼叫，預期耗時數十秒，回應模式（同步 / 非同步）於分析功能切片拍板。
- **Security Requirements:**
  - 分析 API 以 API key 鑑權；僅「已啟用」的 API key 可呼叫。
  - 建立 API key 的端點**公開**，使用者只需提供 API key 名稱；新建立的 API key **預設停用**。
  - 啟用 API key 由 administrator **直接修改資料庫**，不提供管理 API。
  - 另有撤銷 API key 的端點。
  - API key 儲存方式（是否只存雜湊、明文是否只在建立時回傳一次）TBD。
- **Operational Constraints:**
  - AI 成本控制：**同標的在短時間內重用既有分析結果**，不重跑 AI；重用時間窗 TBD。不設每把 key 的次數上限。
  - 部署目標 TBD，目前本機 `go run`。
- **Known Technical Debt / Risks:**
  - 公開的建立 API key 端點可能被大量灌單；目前只靠「預設停用」把關，未做限流。
  - 新聞來源的免費額度與使用條款可能變動。

---

## 6. Glossary Reference

> Domain terms are maintained in `.sdd/UL-MAP.md`. This section links key terms
> relevant to project-level decisions.

| Term | Short Definition |
| :--- | :--- |
| 標的 | 被分析的股票或加密貨幣代號，需搭配市場類別 |
| 市場類別 | `crypto` / `twStock` / `usStock`，決定新聞來源 |
| 新聞來源 | 可爬取新聞的外部平台，由程式依市場類別選擇 |
| 分析事件 | 一次分析的執行紀錄（何時、狀態） |
| 分析結果 | 分析事件的標準化產出（評等、理由、佐證…） |
| 評等 | AI 對標的資訊面的結論 |
| 分析時價格 | 分析當下價格，回測驗證評等的依據 |
| API key | 使用者呼叫分析的憑證，需經 administrator 啟用 |
| Administrator | 啟用 API key 的人（專案擁有者） |
