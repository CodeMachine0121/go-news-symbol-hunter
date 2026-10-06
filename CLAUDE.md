# CLAUDE.md

本檔只做一件事：**告訴你在什麼情境下該去讀哪一份文件。**

- 實作規範（架構、命名、風格、測試）在 `.claude/rules/`，與產品無關。
- 產品知識（專案介紹、技術選型、環境變數、API 路由、領域詞彙）在 `.sdd/`，不要塞進這裡。
- 大小寫、檔名格式、lint 工具一律 follow Go 自身的慣例。

## 規則優先序

**本專案的 `CLAUDE.md` 與 `.claude/rules/` 優先於任何上層目錄的 `CLAUDE.md`。**

上層目錄（如 `~/workspace/CLAUDE.md`）可能殘留其他專案的產品內容與舊版規範（例如「計算行為掛在 entity 上」、`domain/entity/` 資料夾結構），與本專案規則衝突時一律以本專案為準，也不要把那些產品詞彙帶進來。

## 動工前必讀

任何時候要寫或改程式碼，先讀這兩份：

- [.claude/rules/architecture.md](.claude/rules/architecture.md) — 東西該放哪一層、哪個資料夾
- [.claude/rules/naming.md](.claude/rules/naming.md) — 東西該叫什麼名字

## 專案文件（Spec-Driven Development）

| 我想知道 | 去讀 |
| :--- | :--- |
| 專案願景、技術選型、環境變數、API 路由 | `.sdd/PROJECT.md` |
| 某個實體 / 動作 / 識別字該叫什麼 | `.sdd/UL-MAP.md`（**命名以此為準，不得自創同義詞**） |
| 某個功能切片的需求與驗收條件 | `.sdd/{YYYY-MM-DD}-{feature-slug}/BRIEF.md`、`PRD.md` |
| 某個功能切片的技術設計 | `.sdd/{YYYY-MM-DD}-{feature-slug}/ARCH.md` |

- 上述文件尚未建立時，不要假設它們存在；技術選型未定前不要自行決定框架 / ORM / 資料庫。
- **新增功能開新的 `.sdd/{date}-{feature-slug}/` 資料夾**，不要回頭改舊切片的 PRD。
- 修改業務邏輯時同步更新 UL-MAP 與對應 PRD，文件與程式碼不可漂移。

## 開發流程

- 節奏是 **SDD + TDD**：BRIEF → PRD → ARCH → 依驗收條件逐案 red-green-refactor，一個 commit 對應一個小步驟。
- commit 前自行執行並確認全過（目前沒有 pre-commit hook 擋）：
  - `go build ./...`
  - `go vet ./...`
  - `go test ./...`
- 原始碼放 `internal/`，入口放 `cmd/`；組裝根（DI、設定讀取、路由註冊）是唯一認識所有具體型別的地方。

## 情境對照表

| 我正在做的事 | 去讀 |
| :--- | :--- |
| 決定一段程式碼要放哪一層 / 哪個資料夾 | [architecture.md](.claude/rules/architecture.md) |
| 新增 entity、domain model、DTO、VO | [architecture.md](.claude/rules/architecture.md)、[naming.md](.claude/rules/naming.md) |
| 想把業務邏輯寫進 entity | [architecture.md](.claude/rules/architecture.md)（Entity 保持乾淨，行為放 Domain Model） |
| 出現 `private static` 或只被一處使用的 `private` method | [architecture.md](.claude/rules/architecture.md)（三步搬家 / inline 門檻） |
| 想抽一個 method 只為了讓 `defer` 在它結尾還鎖 / 還交易 / 關連線 | [architecture.md](.claude/rules/architecture.md)（inline 門檻的唯一例外：圈住資源持有範圍） |
| 需要一個物件拿著取消函式、完成訊號等「一件正在跑的工作」 | [architecture.md](.claude/rules/architecture.md)（goroutine 生命週期物件不是 Domain Model，住在 service 旁邊、不加後綴） |
| 想在 model 上開 `static`（工廠、`fromXxx` 轉換） | [architecture.md](.claude/rules/architecture.md)（model 內一律不得有 static；轉換寫在來源的 `toXxx()`） |
| 幫 Domain Model 或 VO 命名 | [naming.md](.claude/rules/naming.md)（`Domain` / `Vo` 後綴） |
| 想為 Domain Model 定介面或做繼承 | [architecture.md](.claude/rules/architecture.md)（Domain Model 是由 entity 轉換而來的普通 class，不是介面抽象） |
| 想用 `interface` 描述一份資料 | [naming.md](.claude/rules/naming.md)（介面只抽象行為；資料一律 class） |
| Service 要收一組參數 | [naming.md](.claude/rules/naming.md)（封裝成 DTO，不抽介面、不用行內物件型別） |
| 想開 `XxxHelper`、`XxxUtils` 靜態工具類 | [code-style.md](.claude/rules/code-style.md)（原則禁止；不得已才放 `utilities/`） |
| 幫任何類別 / 介面 / 檔案命名 | [naming.md](.claude/rules/naming.md) |
| 要串接外部 API / 第三方服務（新聞來源、AI） | [naming.md](.claude/rules/naming.md)（一律 `Proxy`；介面用能力抽象命名，不綁供應商） |
| 定義介面（interface） | [naming.md](.claude/rules/naming.md)（`I` 前綴、一介面一檔、以能力命名） |
| 存取資料庫、改資料表結構、寫 migration | [persistence.md](.claude/rules/persistence.md)（一律 Code First、schema sync 交給 ORM） |
| 想手寫 SQL 字串 | [persistence.md](.claude/rules/persistence.md)（禁止） |
| 寫測試、需要 mock 東西 | [testing.md](.claude/rules/testing.md)（只測業務行為；只用 mocking 套件 mock 介面，禁手寫 Fake） |
| 寫排程 / 背景工作、記錄一次執行的狀態 | [background-jobs.md](.claude/rules/background-jobs.md) |
| 選型別、宣告變數、處理金額、包錯誤 | [code-style.md](.claude/rules/code-style.md) |

完整索引見 [.claude/rules/README.md](.claude/rules/README.md)。

## 不可妥協的幾條

即使沒讀完全部規則，這幾條一律成立：

1. **依賴方向永遠指向 Domain**，Domain 不認識 HTTP / ORM / 任何 SDK（包含 AI SDK）。
2. **Entity 是乾淨的 Data Model**，業務行為放 Domain Model。
3. **行為住在它操作的資料旁邊**——沒有 `private static`、沒有 static helper class。
4. **model 內沒有任何 `static`**。轉換一律寫在來源身上：`a.toB()`，不是 `B.fromA(a)`。
5. **Domain Model 不是介面抽象**，是由 entity 轉換而來的普通 class；`Domain` / `Vo` 後綴一眼可辨。
6. **`interface` 只抽象行為，不抽象資料**。所有 data model 一律 `class`；Service 收的參數用 DTO。
7. **介面以「能力」命名，不以「供應商」命名。**
8. **一律 Code First**，schema 交給 ORM，不手寫 SQL / DDL。
9. **測試只驗業務行為**，mock 只用套件 mock 介面。
10. **AI 回傳一律要求結構化 JSON**，由 Domain Model 建構子正規化（非法 enum → 安全預設值、數值 clamp），不信任原始值。
