# 📔 Ubiquitous Language Map

**Project:** go-symbol-news-hunter
**Bounded Context:** 標的資訊面分析（Symbol News Analysis）——依標的與市場類別爬取新聞，交由 AI 分析並產出標準化評等
**Maintainer:** James Hsueh
**Last Updated:** 2026-10-07

> 尚無程式碼，本版詞彙全部來自需求討論。使用者親口定義的詞標 `Confirmed`；討論中提出、方向已同意但細節待 `/clarify` 拍板的標 `Archeology`。

---

## 1. Nouns & Concepts

| Domain Term | Technical Name | User-Facing Label | Definition & Business Rules | Status |
| :--- | :--- | :--- | :--- | :--- |
| 標的 | `Symbol` / `symbol` | symbol | 被分析的股票或加密貨幣代號。同一標的在不同市場寫法不同（`2330`、`AAPL`、`BTC`），一律搭配市場類別才有意義 | Confirmed |
| 市場類別 | `Category` / `category` | category | 決定要爬哪些新聞來源。只允許 `crypto`、`twStock`、`usStock` 三種（見 §4） | Confirmed |
| 分析事件 | `AnalysisEvent` / table `analysis_events` | analysisEventId | 記錄「什麼時候執行了一次分析」：發起的 API key、標的、市場類別、狀態、失敗原因、開始 / 結束時間、AI 模型與用量；不含分析結論 | Confirmed |
| 分析結果 | `AnalysisResult` / table `analysis_results` | result | 一次分析事件的標準化產出，以 `analysis_event_id` 關聯所屬分析事件（一對一）；含 `created_at` | Confirmed |
| 評等 | `Grade` / `grade` | grade | AI 對標的資訊面的結論評等，值域見 §4，非法值正規化為中性 | Confirmed |
| 分析理由 | `Reason` / `reason` | reason | 支撐評等的文字說明 | Confirmed |
| 新聞來源 | `NewsProvider`（介面 `INewsProxy`，實作 `{Provider}NewsProxy`） | — | 可爬取新聞的外部平台。依市場類別選用，由程式決定，不交給 AI 選擇 | Confirmed |
| 新聞 | `News` | news | 由新聞來源取回、正規化後的單則新聞：標題、連結、發布時間、新聞來源名稱、摘要（來源有提供時） | Confirmed |
| 搜尋字 | `SearchKeyword` | — | 向新聞來源搜尋時使用的字：台股為公司簡稱、美股為標的代號、加密貨幣為幣種名稱（專門來源另以名稱或代號篩選） | Confirmed |
| 標的辨識 | `ResolvedSymbol` | — | 依市場類別把標的轉為搜尋字的結果；台股依序查證交所上市公司清單、櫃買中心上櫃股票資料，加密貨幣查幣種清單（同代號取市值排名最前），美股不查 | Confirmed |
| 上櫃股票 | — | 上櫃 | 在櫃買中心掛牌交易的股票，屬市場類別台股；公司簡稱與收盤價取自櫃買中心「上櫃股票每日收盤行情」 | Confirmed |
| 新聞搜尋時間窗 | — | — | 只保留搜尋當下往回 7 天內發布的新聞 | Confirmed |
| 失敗的新聞來源 | `FailedNewsProviders` | failedNewsProviders | 本次搜尋未取得資料的新聞來源名稱清單 | Confirmed |
| 信心指數 | `Confidence` / `confidence` | confidence | AI 對評等的把握程度，0–100，超出範圍 clamp | Confirmed |
| 時間範圍 | `TimeHorizon` / `time_horizon` | time_horizon | 評等適用的持有期間（候選 `short`/`mid`） | Confirmed |
| 關鍵事件 | `KeyEvent` / `key_events` | key_events | 支撐評等的 3–5 則重點事件，每則附來源 URL 與發布時間 | Confirmed |
| 風險因子 | `RiskFactor` / `risk_factors` | risk_factors | 與主結論方向相反的訊號 | Confirmed |
| 分析時價格 | `PriceAtAnalysis`（`Price`、`PriceCurrency`、`PricedAt`、`PriceSource`） | priceAtAnalysis | 分析完成時依市場類別取得的價格；精確小數；必須大於 0，取不到時為「未取得」（null），不影響分析完成。專案成功指標「評等可回測驗證」的基準點 | Confirmed |
| 價格來源 | `PriceSource` | priceSource | 加密貨幣 Binance（對 USDT）、美股 Yahoo 財經、台股上市為證交所、上櫃為櫃買中心（皆為最近交易日收盤） | Confirmed |
| 價格時間 | `PricedAt` | pricedAt | 報價所屬的時間：Binance 為取得時間、Yahoo 財經為來源報價時間、證交所為收盤日期（台北時間） | Confirmed |
| 分析佐證 | `Evidence` / `evidence` | — | 當次餵給 AI 的新聞清單快照（JSON），存於分析結果，供稽核 AI 為何如此評等 | Confirmed |
| 分析事件狀態 | `AnalysisEventStatus` / `status` | status | 分析事件的執行狀態（見 §4） | Confirmed |
| API key | `ApiKey`（table 名 TBD） | API key | 使用者呼叫分析的憑證。建立時由使用者提供名稱，**預設停用**；僅已啟用者可呼叫分析 | Confirmed |
| API key 名稱 | `Name` / `name` | name | 使用者建立 API key 時必填的名稱 | Confirmed |
| 啟用狀態 | `IsActive` / `is_active` | is_active | API key 是否可用。預設 `false`，由 administrator 改為 `true` 才算啟用 | Confirmed |
| Administrator | — | administrator | 專案擁有者；以直接修改資料庫的方式啟用 API key，不提供管理 API | Confirmed |
| 撤銷時間 | `RevokedAt` / `revoked_at` | — | API key 被撤銷的時間；有值即「已撤銷」，不可逆，且優先於啟用狀態 | Confirmed |
| API key 狀態 | `ApiKeyStatus` | status | 對外呈現的狀態：停用中 / 已啟用（見 §4）；已撤銷的 API key 對外一律視為「無效」 | Confirmed |
| 受保護功能 | — | — | 需要已啟用且未撤銷的 API key 才能使用的功能（如分析標的） | Confirmed |
| 重用時間窗 | `AnalysisReuseWindow` | — | 6 小時。同標的 + 市場類別有分析中的分析事件、或 6 小時內完成的分析事件時，直接回傳該分析事件、不重跑 AI | Confirmed |
| AI 用量 | `InputTokens` / `OutputTokens` | — | 一次分析所有 AI 呼叫的輸入、輸出量合計，記在分析事件 | Confirmed |
| 失敗原因 | `FailureReason` / `failure_reason` | failureReason | 分析事件失敗的原因文字（見 §4） | Confirmed |

---

## 2. Actions & Processes

| Business Action | Technical Method | Trigger | Business Impact | Notes |
| :--- | :--- | :--- | :--- | :--- |
| 搜尋標的新聞（爬取新聞） | `SearchSymbolNews` | 使用者以已啟用 API key 呼叫；之後亦由 AI 分析時呼叫 | 依市場類別同時向對應新聞來源取新聞，篩選 7 天內、標題去重、新到舊、最多 30 則，不落地 | 部分新聞來源失敗仍回傳其餘結果並列出失敗來源 |
| 選擇新聞來源 | `NewsProviderSelection` | 搜尋標的新聞時 | 台股 → 鉅亨網、Google 新聞（繁中）；美股 → Yahoo 財經、Google 新聞（英文）；加密貨幣 → CoinDesk、Cointelegraph、Google 新聞（英文） | 由程式決定，不由 AI 決定 |
| 發起分析 | `StartSymbolAnalysis` | 使用者以已啟用 API key 呼叫（標的、市場類別） | 驗證並辨識標的 → 重用或建立分析事件 → 背景執行分析 | 非同步：立即回傳分析事件 |
| 執行分析 | `AnalyzeSymbol` | 新建立的分析事件，於背景執行 | AI 以搜尋標的新聞取得新聞 → 提交結論 → 正規化 → 保存分析結果、分析事件改為已完成；失敗則記錄原因 | 最多 5 輪工具使用 |
| 查詢分析事件 | `GetAnalysisEvent` | 使用者以已啟用 API key 查詢編號 | 回傳狀態、失敗原因或分析結果 | 不限發起者 |
| 取得分析時價格 | `FetchPrice` | 分析結論通過正規化後 | 依市場類別向價格來源取價；失敗則記為未取得 | 最多等待 10 秒 |
| 中斷分析事件 | `FailInterruptedAnalysisEvents` | 服務啟動時 | 仍為分析中的分析事件改為失敗 | 原因「服務重新啟動，分析中斷」 |
| 建立 API key | 待定 | 使用者呼叫公開的建立端點（必填名稱） | 新增一把停用中的 API key | 端點公開，靠預設停用把關；明文是否只回傳一次 TBD |
| 撤銷 API key | 待定 | 持有者出示 API key 撤銷 | API key 永久失效 | 持有 API key 即可撤銷；撤銷後不可再啟用 |
| 查詢 API key 狀態 | 待定 | 使用者出示 API key 查詢 | 回傳名稱與狀態，不回傳完整 API key | 已撤銷 / 不存在 → 無效 |
| 驗證 API key | 待定 | 每次呼叫受保護功能 | 未提供 → 需要提供；不存在或已撤銷 → 無效；停用中 → 尚未啟用；其餘放行 | |
| 啟用 API key | —（直接改資料庫） | Administrator 手動操作 | API key 的啟用狀態改為 `true` | 不提供 API |
| 重用分析結果 | 待定 | 分析標的時，重用時間窗內已有同標的分析結果 | 直接回傳既有分析結果，不建立新的 AI 分析 | 控制 AI 成本 |
| 標準化分析輸出 | 待定 | AI 回傳後 | AI 原始 JSON 經 Domain Model 建構子正規化（非法 enum → 安全預設值、數值 clamp）後成為分析結果 | 不信任 AI 原始值 |

---

## 3. Ambiguities & Conflicts

| Ambiguous Term | Meaning in Context A | Meaning in Context B | Resolution |
| :--- | :--- | :--- | :--- |
| 標的代號格式 | 台股：數字代號 `2330`，但新聞多寫公司名「台積電」 | Crypto：`BTC` / `BTCUSDT` / `bitcoin` 多種寫法 | 已決議：標的去空白；加密貨幣與美股轉大寫；以「標的辨識」轉為搜尋字（見 §1） |
| category 的值 | 使用者原文 `twSotck` | 討論中寫作 `twStock` | 以 `twStock` 為準（原文為筆誤，待確認） |
| 分析事件 vs 分析結果 | 分析事件：執行過程（何時、狀態、失敗原因） | 分析結果：業務產出（評等、理由…） | 兩者分責；目前一個分析事件對應一個分析結果 |
| 資料表名稱 | 使用者定義為單數 `analysis_event` / `analysis_result` | ORM 預設會轉為複數（如 GORM → `analysis_events`） | 已決議：依 persistence 規則沿用 ORM 預設命名（`analysis_events`、`analysis_results`），不覆寫 |
| 撤銷 vs 停用 | 撤銷：使用者主動作廢 API key | 停用：啟用狀態為 `false`（含尚未經 administrator 啟用） | 已決議：撤銷是獨立且不可逆的狀態（撤銷時間），優先於啟用狀態；administrator 停用則可再啟用 |
| 新聞來源 vs provider | 中文「新聞來源」 | 英文 provider；命名規範要求外部資源以 `Proxy` 結尾 | 業務詞用「新聞來源」；程式介面以能力命名 `INewsProxy`，實作帶供應商前綴 |

---

## 4. External & Enum Mapping

| Category | Code Value / Key | Domain Label | Description |
| :--- | :--- | :--- | :--- |
| 市場類別 | `crypto` | 加密貨幣 | Confirmed |
| 市場類別 | `twStock` | 台股（上市 + 上櫃；興櫃不支援） | Confirmed |
| 市場類別 | `usStock` | 美股 | Confirmed |
| 評等 | `strongBullish` / `bullish` / `neutral` / `bearish` / `strongBearish` | 強烈看多 / 看多 / 中性 / 看空 / 強烈看空 | Confirmed；非法值正規化為 `neutral` |
| 分析事件狀態 | `running` / `succeeded` / `failed` | 分析中 / 已完成 / 失敗 | Confirmed；服務重啟時殘留 `running` 改為 `failed` |
| 失敗原因 | — | AI 服務暫時無法使用 / AI 未在限制內完成分析 / AI 未提供完整分析 / AI 拒絕分析此標的 / 分析結果保存失敗 / 服務重新啟動，分析中斷 / 分析逾時 | Confirmed |
| API key 狀態 | `inactive` / `active` | 停用中 / 已啟用 | Confirmed；已撤銷不作為對外狀態，一律回「無效」 |
| 時間範圍 | `short` / `mid` | 短期（兩週內）/ 中期（三個月內） | Confirmed；非法值正規化為 `short` |

---

## Quick Start Guide
1. **Archeology** — read source code; fill `Technical Name` with raw names found in the codebase.
2. **Mapping** — check UI screens or ask business stakeholders; fill `Domain Term` with the correct canonical name.
3. **Refine** — add business rules (e.g., "this field cannot be negative", "this action must occur after checkout").
4. **Sync** — this document is the single authoritative dictionary for all future renaming, refactoring, and new documentation.
