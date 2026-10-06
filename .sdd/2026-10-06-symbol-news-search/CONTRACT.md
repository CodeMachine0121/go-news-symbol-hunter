# Contract Traceability Matrix — 標的新聞搜尋（symbol-news-search）

Contract: PRD.md（`.sdd/2026-10-06-symbol-news-search/PRD.md`，v1.0 Finalized）
Design map: ARCH.md（同資料夾）
Glossary: `.sdd/UL-MAP.md`
Implementation: `internal/`、`cmd/server/`（branch `feature/symbol-news-search`）
Oracle: Acceptance Criteria（31 clauses：AC × 21、BR × 8、NFR × 2）
Audit date: 2026-10-06

> **Ceiling：** 這是靜態一致性稽核。各條款的預期結果（oracle）先只從 PRD 推導，再分別判斷「測試是否斷言該結果」與「正式程式碼是否產生該結果」；不撰寫新的探測、不執行自創情境。僅執行既有、已對應到條款的單一測試做佐證（全部通過，但判決不以 pass/fail 為依據）。另以 scratchpad 內獨立小程式驗證 Go `encoding/xml`／`encoding/json` 對「格式正確但結構不符」內容的行為（不觸及專案）。

**Oracle → 具體產物的橋接（Phase 3 第 0 步，依 ARCH §3 介面細節與 UL-MAP）：**

| 業務結果 | 具體產物 |
| :--- | :--- |
| 搜尋被拒，告知「API key 尚未啟用」 | 403、`api_key_inactive`、message `API key 尚未啟用` |
| 搜尋被拒，告知「需要提供 API key」 | 401、`api_key_missing`、message `需要提供 API key` |
| 「市場類別只能是 crypto、twStock、usStock」 | 400、`market_category_unsupported`（`ErrMarketCategoryUnsupported`） |
| 「標的為必填」 | 400、`symbol_required`（`ErrSymbolRequired`） |
| 「找不到此標的」 | 404、`symbol_not_found`（`ErrSymbolNotFound`） |
| 「新聞來源暫時無法使用」 | 502、`news_providers_unavailable`（`ErrNewsProvidersUnavailable`） |
| 告知某新聞來源這次沒有取得資料 | `failedNewsProviders` 含該來源名稱 |
| 新聞清單 | `SymbolNewsDto.News`（`title`/`link`/`publishedAt`/`providerName`/`summary`） |

## Clauses

`Spec-expected` 欄為 Phase 2 僅依 PRD 推導出的業務結果；兩個稽核欄各自對照此 oracle 判斷。

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| AC-1 | US-01 已啟用的 API key 可以搜尋：Given 使用者持有已啟用的 API key / When 搜尋標的 BTC、市場類別加密貨幣 / Then 使用者取得 BTC 的新聞清單 | 已啟用 API key 通過把關，搜尋成功並取得該標的的新聞清單 | `cmd/server/dependencies.go:85-86`、`internal/controller/api_key_controller.go:71-81`、`internal/controller/news_controller.go:27-37` | `internal/controller/tests/news_controller_test.go:60-71`（已啟用 key → 200 + 完整新聞清單；用 2330/twStock，把關與市場無關） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-2 | US-01 停用中的 API key 被拒：Then 搜尋被拒，告知「API key 尚未啟用」 | 搜尋被拒，且出現「API key 尚未啟用」 | `api_key_controller.go:25,71-78`；`dependencies.go:85-86` 把 `/news` 掛在把關之後 | `news_controller_test.go:73-84`（403 + `api_key_inactive` + 訊息） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-3 | US-01 未提供 API key 被拒：Then 搜尋被拒，告知「需要提供 API key」 | 搜尋被拒，且出現「需要提供 API key」 | `api_key_controller.go:23,71-78`；`dependencies.go:85-86` | `news_controller_test.go:73-84`（401 + `api_key_missing` + 訊息）；`cmd/server/dependencies_test.go:22-46`（實際 `registerRoutes` 下 `/news` 無 key → 401） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-4 | US-02 不支援的市場類別被拒：Given 市場類別「港股」 / Then 告知「市場類別只能是 crypto、twStock、usStock」 | 搜尋被拒，出現「市場類別只能是 crypto、twStock、usStock」 | `internal/domain/models/vo/market_category_vo.go:11,17-24`；`news_controller.go:13` | `news_controller_test.go:95`（`hk` → 400 + 訊息）；`internal/domain/models/vo/tests/market_and_symbol_vo_test.go:21`（`港股`）；`internal/application/tests/news_search_application_test.go:147` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-5 | US-02 未提供市場類別被拒：Then 告知「市場類別只能是 crypto、twStock、usStock」 | 同上訊息被拒 | `market_category_vo.go:17-24`（空字串落入 default） | `news_controller_test.go:96`；`market_and_symbol_vo_test.go:22`；`news_search_application_test.go:148` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-6 | US-02 未提供標的被拒：Given 沒有提供標的，或標的只有空白 / Then 告知「標的為必填」 | 未提供或只有空白都被拒，出現「標的為必填」 | `internal/domain/models/vo/symbol_vo.go:15-19`；`news_controller.go:14` | `news_controller_test.go:97`（未提供 → 400 + 訊息）；`market_and_symbol_vo_test.go:45-46`（空字串、全空白）；`news_search_application_test.go:149`（`" "`） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-7 | US-02 標的去除空白並轉為大寫：Given 標的「 aapl 」、美股 / Then 系統以 AAPL 搜尋 / And 結果中的標的顯示為 AAPL | 向新聞來源搜尋時用 AAPL，且結果標的為 AAPL | `symbol_vo.go:16-23`；`internal/domain/service/symbol_resolution_service.go:39-40`；`internal/domain/service/news_search_service.go:51,74` | `news_search_application_test.go:86-96`（mock 僅接受 `FetchNews("AAPL")`、斷言 `Symbol == "AAPL"`）；`market_and_symbol_vo_test.go:42` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-8 | US-02 不存在的台股標的被拒：Given 證交所上市公司中沒有 9999 / Then 告知「找不到此標的」 | 搜尋被拒，出現「找不到此標的」 | `symbol_resolution_service.go:21-28`；`internal/infrastructure/twse/twse_listed_company_proxy.go:53-54`；`news_controller.go:15` | `news_controller_test.go:98-100`（404 + 訊息）；`news_search_application_test.go:150-152`；`internal/infrastructure/twse/tests/twse_listed_company_proxy_test.go:40,47` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-9 | US-02 無法辨識的加密貨幣被拒：Given 沒有任何幣種代號是 NOTACOIN / Then 告知「找不到此標的」 | 搜尋被拒，出現「找不到此標的」 | `symbol_resolution_service.go:30-37`；`internal/infrastructure/coingecko/coin_gecko_cryptocurrency_proxy.go:59-75` | `news_search_application_test.go:153-155`（`ErrSymbolNotFound`，與 AC-8 同一哨兵 → 同一 404 訊息）；`internal/infrastructure/coingecko/tests/coin_gecko_cryptocurrency_proxy_test.go:43` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-10 | US-02 查無新聞的美股回傳空清單：Given 所有美股新聞來源都沒有 ZZZZ 的新聞 / Then 搜尋成功，新聞清單為空 | 搜尋成功（非拒絕），新聞清單為空 | `symbol_resolution_service.go:39-40`（美股不辨識）；`news_search_service.go:61-78`；`internal/domain/models/domains/news_collection_domain.go:63-64` | `news_search_application_test.go:98-108`（無錯誤、`News == []`） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-11 | US-03 台股以公司簡稱向鉅亨網與 Google 新聞（繁中）搜尋：Given 2330 簡稱「台積電」 / Then 系統以「台積電」向鉅亨網與 Google 新聞（繁體中文）取得新聞 | 以「台積電」查詢鉅亨網，並以「台積電」查詢**繁體中文**版 Google 新聞 | `symbol_resolution_service.go:21-29`；`dependencies.go:45,48-51`（twStock 綁 `GoogleNewsTraditionalChineseLocale`）；`internal/infrastructure/news/cnyes_news_proxy.go:42-43`；`internal/infrastructure/news/google_news_proxy.go:34-42` | `news_search_application_test.go:70-84`（兩來源都收到「台積電」）；`dependencies_test.go:65`（只比對來源名稱）；`internal/infrastructure/news/tests/news_proxy_test.go:62-73`（`GoogleNewsProxy` 單獨用繁中 locale 時的參數） | shallow — 沒有任何測試斷言「台股那一列的 Google 新聞是繁體中文」：`dependencies_test.go:51-69` 只比 `ProviderName()`，把 `dependencies.go:50` 換成英文 locale 全部測試仍會通過 | produces-oracle | 🟠 mis-asserted |
| AC-12 | US-03 美股以代號向 Yahoo 財經與 Google 新聞（英文）搜尋：Then 系統以 AAPL 向 Yahoo 財經與 Google 新聞（英文）取得新聞 | 以 AAPL 查詢 Yahoo 財經與**英文**版 Google 新聞 | `dependencies.go:46,52-55`；`internal/infrastructure/news/yahoo_finance_news_proxy.go:22-25`；`google_news_proxy.go:34-42` | `news_search_application_test.go:86-96`；`dependencies_test.go:66`；`news_proxy_test.go:75-83,85-96` | shallow — 同 AC-11：組裝根「美股那列 Google 新聞為英文」無測試斷言 | produces-oracle | 🟠 mis-asserted |
| AC-13 | US-03 加密貨幣以幣種名稱與代號篩選 CoinDesk、Cointelegraph，並向 Google 新聞（英文）搜尋：Given BTC → Bitcoin / Then 以 Bitcoin 向 Google 新聞（英文）搜尋 / And CoinDesk 與 Cointelegraph 只保留標題或摘要提到「Bitcoin」或「BTC」的新聞 | 以 Bitcoin 查詢**英文**版 Google 新聞；CoinDesk 與 Cointelegraph 僅留下標題或摘要提及 Bitcoin 或 BTC 的新聞；Google 新聞不被此規則篩掉 | `symbol_resolution_service.go:30-38`；`internal/domain/models/vo/resolved_symbol_vo.go:13-15`；`news_search_service.go:53-55`；`news_collection_domain.go:25-36`；`dependencies.go:46,56-60` | `news_search_application_test.go:110-124`（Google 收到 Bitcoin、CoinDesk 被篩、Google 未被篩；fixture 未含 Cointelegraph）；`internal/domain/models/domains/tests/news_collection_domain_test.go:24-34`（標題含 Bitcoin、摘要含 btc 皆保留）；`dependencies_test.go:67-68`（CoinDesk/Cointelegraph 篩選旗標為 true） | shallow — 「加密貨幣那列 Google 新聞為英文」無測試斷言（同 AC-11）；其餘部分有斷言 | produces-oracle | 🟠 mis-asserted |
| AC-14 | US-04 只保留最近 7 天內的新聞：6 天 23 小時前保留、7 天 1 小時前排除 | 結果只含 6 天 23 小時前那則 | `news_collection_domain.go:42-46` | `news_collection_domain_test.go:36-46` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-15 | US-04 相同標題的新聞只保留一則：鉅亨網與 Google 新聞皆有「台積電法說會」 | 結果中「台積電法說會」只有一則 | `news_collection_domain.go:50-58` | `news_collection_domain_test.go:48-56`；`news_search_application_test.go:70-84` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-16 | US-04 依發布時間由新到舊排序：前天、今天、昨天 → 今天、昨天、前天 | 結果順序為今天、昨天、前天 | `news_collection_domain.go:47-49` | `news_collection_domain_test.go:67-75` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-17 | US-04 最多回傳 30 則：符合條件 45 則 → 只含最新的 30 則 | 回傳恰 30 則，且是最新的 30 則 | `news_collection_domain.go:14,54` | `news_collection_domain_test.go:77-88`（筆數 30、第一則最新、第 30 則為第 30 新） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-18 | US-04 每則新聞包含完整資訊：鉅亨網附摘要的新聞 → 標題、連結、發布時間、新聞來源名稱「鉅亨網」與摘要 | 該則新聞帶有標題、連結、發布時間、來源名稱「鉅亨網」、摘要 | `cnyes_news_proxy.go:52-59`；`news_collection_domain.go:63-75`；`internal/domain/models/dto/news_dto.go:5-11` | `news_proxy_test.go:49-60`（五欄位皆斷言、來源名稱「鉅亨網」）；`news_collection_domain_test.go:90-94`；`news_controller_test.go:60-71`（JSON 五欄位） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-19 | US-05 單一新聞來源失敗仍回傳其他結果：Google 新聞無回應、鉅亨網正常 → 搜尋成功，含鉅亨網新聞 / 告知 Google 新聞這次沒有取得資料 | 搜尋成功；結果含鉅亨網新聞；並告知 Google 新聞本次未取得資料 | `news_search_service.go:61-78` | `news_search_application_test.go:126-137`；`news_controller_test.go:60-71`（`failedNewsProviders: ["Google 新聞"]`） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-20 | US-05 所有新聞來源都失敗 → 告知「新聞來源暫時無法使用」 | 搜尋被拒，出現「新聞來源暫時無法使用」 | `news_search_service.go:70-72`；`news_controller.go:16` | `news_controller_test.go:101-105`（502 + 訊息）；`news_search_application_test.go:162-165` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-21 | US-05 辨識標的所需的資料取不到：證交所上市公司清單暫時取不到 → 告知「新聞來源暫時無法使用」 | 搜尋被拒，出現「新聞來源暫時無法使用」 | `symbol_resolution_service.go:23-25`（`%w` 包裝）；`internal/controller/error_response_table.go:20-21`（`errors.Is`，訊息取哨兵本身）；`twse_listed_company_proxy.go:38-45` | `news_search_application_test.go:156-158`（`ErrorIs ErrNewsProvidersUnavailable`）；`twse_listed_company_proxy_test.go:66-86` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-1 | 市場類別與新聞來源對應：台股 → 鉅亨網、Google 新聞（繁中）；美股 → Yahoo 財經、Google 新聞（英文）；加密貨幣 → CoinDesk、Cointelegraph、Google 新聞（英文） | 三個市場各自使用且只使用上列來源，Google 新聞語系依市場為繁中／英文／英文 | `dependencies.go:43-62` | `dependencies_test.go:48-70` | shallow — 來源名稱與順序有斷言，但 Google 新聞語系（繁中 vs 英文）未斷言 | produces-oracle | 🟠 mis-asserted |
| BR-2 | 「同一則新聞」：標題去除前後空白、不分大小寫後相同；保留先出現（較新）的那則 | 前後空白／大小寫不同的同標題視為同一則，只留較新那則 | `news_collection_domain.go:47-58`（先穩定排序新到舊，再以 trim+lower 去重） | `news_collection_domain_test.go:48-56`（含前後空白、保留較新的 Google 新聞）、`:58-65`（大小寫） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-3 | 7 天以「搜尋當下」往回推算，發布時間剛好 7 天前（含）以內的新聞保留 | 恰好 7 天前發布的新聞保留；超過 7 天的排除 | `news_collection_domain.go:43-46`（`Before(publishedSince)` 才刪除）；`news_search_service.go:76`（以 clock 當下時間） | `news_collection_domain_test.go:36-46`（`exactly seven days ago` 保留） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-4 | 加密貨幣代號對應多個幣種時，取市值排名最前者 | 同代號多幣種時採市值排名最前者的名稱 | `coin_gecko_cryptocurrency_proxy.go:59-73` | `coin_gecko_cryptocurrency_proxy_test.go:32-59` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-5 | 證交所上市公司清單與幣種辨識結果保留 24 小時後重新取得 | 24 小時內重用、滿 24 小時重新取得 | `twse_listed_company_proxy.go:13,37,51`；`coin_gecko_cryptocurrency_proxy.go:14,48,59,74` | `twse_listed_company_proxy_test.go:50-64`；`coin_gecko_cryptocurrency_proxy_test.go:61-77` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-6 | Edge：單一新聞來源超過 10 秒未回應視為失敗 | 某來源 10 秒內未回應 → 該來源列為失敗，其餘照常 | `dependencies.go:24,66`（`http.Client{Timeout: 10s}`，涵蓋讀取 body）；`internal/utilities/http_body_reader.go:23-31` | — 沒有測試斷言 10 秒逾時（`http_body_reader_test.go` 只測非 2xx／連不到；組裝根逾時常數未被斷言） | no-test | produces-oracle | 🟡 partial |
| BR-7 | Edge：新聞來源回傳無法辨識的發布時間 → 該則新聞略過 | 該則略過，同來源其他新聞照常回傳 | `internal/utilities/rss_feed_parser.go:51-54`；鉅亨網缺漏或 0 的 `publishAt` 轉為 1970 年後被 7 天窗排除（`cnyes_news_proxy.go:56`、`news_collection_domain.go:43-46`） | `internal/utilities/tests/rss_feed_parser_test.go:12-26`（`yesterday` 那則被略過、其餘保留） | asserts-oracle | produces-oracle | ✅ conforms（附註見下） |
| BR-8 | Edge：新聞來源回傳的資料格式無法解析 → 視為該新聞來源失敗 | 回應不是該來源應有的格式 → 該來源列為失敗（不是「成功但 0 則」） | `rss_news_reader.go:22-25`→`rss_feed_parser.go:45-48`；`cnyes_news_proxy.go:47-50` | `news_proxy_test.go:119-166`（只測語法錯誤的 `<rss><channel><item>` 與 `{`） | asserts-oracle（就語法錯誤的情況） | diverges — 只有語法錯誤才失敗。`rssDocument`（`rss_feed_parser.go:27-36`）沒有 `XMLName`，任何格式正確的 XML（如 `<error>…</error>`）都解析成 0 則且無錯；鉅亨網任何格式正確的 JSON（如 `{"statusCode":500,"data":null}`）也解析成 0 則且無錯。已以獨立程式確認兩者皆回 `nil` error。結果：來源實際出錯卻被當成功，不會進 `failedNewsProviders`，且可能讓 AC-20「全部失敗」變成「成功、空清單」 | 🔴 violation |
| NFR-1 | Performance：各新聞來源同時查詢；整次搜尋在最慢的新聞來源逾時（10 秒）內完成 | 各來源並行查詢；即使最慢來源逾時，整次搜尋約 10 秒內結束 | 並行：`news_search_service.go:47-59`（`WaitGroup.Go`）。但：標的辨識在並行前**依序**執行，且有自己的 10 秒逾時（`news_search_service.go:41` → `twse_listed_company_proxy.go:38`／`coin_gecko_cryptocurrency_proxy.go:51`）；兩個辨識 proxy 在整段 HTTP 呼叫期間持有 mutex（`twse_listed_company_proxy.go:34-35`、`coin_gecko_cryptocurrency_proxy.go:45-46`，CoinGecko 是跨代號的全域鎖） | — 沒有測試斷言並行或整體耗時上限 | no-test | diverges — 台股／加密貨幣在快取未命中時最壞約 10 秒（辨識）+ 10 秒（新聞）≈ 20 秒；同時進來的請求還會排在持鎖的慢速辨識呼叫之後，等待時間再往上加 | 🔴 violation |
| NFR-2 | Security：沿用 API key 把關 | `/news` 只有已啟用且有效的 API key 能使用 | `dependencies.go:85-86` | `dependencies_test.go:22-46`（實際路由下無 key → 401）；`news_controller_test.go:73-84` | asserts-oracle | produces-oracle | ✅ conforms |

**BR-7 附註：** 鉅亨網回應中若 `publishAt` 不是數字（例如字串），`json.Unmarshal` 會讓整個來源失敗（`cnyes_news_proxy.go:48-50`），而不是只略過那一則。這落在 BR-7（略過該則）與 BR-8（格式無法解析 → 來源失敗）兩條規則的重疊處，PRD 沒有規定哪條優先，因此這裡判為規格模糊、不計為違反。若要採 BR-7，需改成逐則解碼。

## Orphans (code with no clause)

| Code | Description | Verdict |
|------|-------------|---------|
| `internal/controller/error_response_table.go:25` | 未列在對照表的錯誤一律回 503 `service_unavailable`／「服務暫時無法使用」。ARCH 有記載，PRD 沒有；目前 `/news` 不會走到這條路徑（service 只回傳已對照的哨兵錯誤） | undocumented（ARCH-only 行為，PRD 無對應條款） |
| `internal/domain/service/news_search_service.go:33-40` | 市場類別先於標的驗證：同時缺少兩者時只回「市場類別只能是…」，不會提到「標的為必填」。PRD 沒有規定兩個錯誤的優先順序 | undocumented（驗證優先順序未入 PRD） |

已對照 Out of Scope 負面清單：沒有 AI 分析、沒有抓全文（只用標題／摘要）、沒有保存或快取新聞（快取只用於標的辨識，屬 BR-5）、沒有上櫃／興櫃、也沒有讓使用者自訂時間範圍／數量／來源（請求只收 `symbol`、`category`）→ **無越界違反**。

## Summary

- Conforms: 24/31 clauses ✅（77.4%）
- Violations: BR-8、NFR-1（程式碼產生的結果不對）
- Mis-asserted: AC-11、AC-12、AC-13、BR-1（測試通過但沒鎖住 Google 新聞語系與市場的綁定）
- Partial: BR-6（10 秒逾時沒有測試）
- Gaps: —
- Unclear: —
- Orphans: 2（皆為 undocumented，無越界違反）

---

## 稽核後修正紀錄（2026-10-06）

| 項目 | 處理 |
| :--- | :--- |
| BR-8 格式正確但非預期結構被當成成功 0 則 | 修正：RSS 根元素必須為 `rss`；鉅亨網必須有 `data.items`；否則該來源失敗。補測試 |
| NFR-1 整次搜尋可能超過 10 秒 | 標的辨識與取新聞有先後依賴，無法並行；PRD 改為兩段各最多 10 秒（辨識有 24 小時快取）。修正辨識 Proxy 在 HTTP 期間持鎖造成排隊，補併發測試 |
| AC-11、AC-12、AC-13、BR-1 測試未鎖住語系 | 組裝根改用 `ExternalSourceUrls`，補測試斷言各市場 Google 新聞的 `hl`/`gl`/`ceid` |
| BR-6 逾時無測試 | 補測試：外部 HTTP client 逾時為 10 秒；逾時的來源讀取回傳錯誤 |
| O-1 未列出的錯誤回 503 | 保留（ARCH 記載） |
| O-2 市場類別先於標的驗證 | 保留：市場類別決定標的正規化方式，必須先驗證 |
| 鉅亨網 `publishAt` 非數字 | 歸類為 BR-8（格式無法解析 → 來源失敗） |
