# Contract Traceability Matrix — 2026-10-06-symbol-news-analysis（標的資訊面分析）

Contract: PRD.md（v1.0, Finalized）
Design map: ARCH.md（Confirmed）
Glossary: .sdd/UL-MAP.md
Implementation: `internal/`（domain / application / controller / infrastructure）、`cmd/server/`；branch `feature/symbol-news-analysis` @ `fde23ea`
Oracle: Acceptance Criteria（44 clauses：AC 28、BR 12、NFR 4）

> **天花板（Ceiling）：** 本文件為**靜態符合度稽核**。對每條條款先只依 PRD 推導業務可觀察的預期結果（oracle），再分別獨立判斷「測試是否斷言該 oracle」與「正式程式碼是否產生該 oracle」；不撰寫新探針、不執行自行設計的情境。為佐證曾只執行對應條款的既有測試（domain / application / controller / claude proxy，以及以 `TEST_POSTGRES_DSN` 連 port 55433 的 repository 測試，6 個分析相關 repository 測試皆實際執行未 skip），全部通過；但判定一律來自 oracle 比對，不來自綠燈。

## 條款來源與編號

- **AC-1 ~ AC-28**：PRD §3 的每一個 Gherkin `Scenario`（依出現順序）。
- **BR-1 ~ BR-9**：PRD §4 Core Business Rules（「關鍵事件 / 風險因子」一句拆為 BR-4、BR-5）；**BR-10 ~ BR-12**：PRD §4 Edge Cases。
- **NFR-1 ~ NFR-4**：PRD §6（Performance、Security、Cost、Analytics）。
- **Out of Scope（負面檢查清單）**：分析時價格與回測、定時自動分析、分析事件列表 / 依標的查詢歷史、取消分析。

## Oracle 到具體形式的橋接（Phase 3 第 0 步）

依 UL-MAP 與 ARCH：分析中 / 已完成 / 失敗 ↔ `running` / `succeeded` / `failed`；評等 強烈看多…強烈看空 ↔ `strongBullish`/`bullish`/`neutral`/`bearish`/`strongBearish`；時間範圍 短期 / 中期 ↔ `short` / `mid`；分析事件編號 ↔ `analysisEventId`；「發起被拒，告知 X」↔ HTTP 4xx 且 `error.message == X`；「沒有建立分析事件」↔ `IAnalysisEventRepository.Create` 未被呼叫；「重新啟動」↔ `main` 啟動時呼叫 `FailInterruptedAnalysisEvents`；「5 輪工具使用」↔ `MaximumAnalystRounds = 5` 次 `IAnalystProxy.Respond`。

## Clauses

`Spec-expected` 欄為 Phase 2 僅依 PRD 推導的業務 oracle；兩個稽核欄位檢查的是其橋接後的具體產物。

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| AC-1 | 發起分析成功：已啟用 key、BTC（加密貨幣）近 6 小時無分析事件 → 取得新的分析事件編號，狀態「分析中」 | 建立一筆新分析事件並回傳其編號，該事件狀態為分析中 | `internal/domain/service/symbol_analysis_service.go:56-64`；`internal/controller/analysis_event_controller.go:48-52` | `internal/application/tests/symbol_analysis_application_test.go:67-99`；`internal/controller/tests/analysis_event_controller_test.go:57-79` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-2 | 不支援的市場類別不建立分析事件：市場類別「港股」→ 發起被拒，告知「市場類別只能是 crypto、twStock、usStock」，且沒有建立任何分析事件 | 發起被拒並顯示「市場類別只能是 crypto、twStock、usStock」；不產生任何分析事件 | `internal/domain/service/symbol_resolution_service.go:21-24`（在 `symbol_analysis_service.go:39-42` 建立前即返回）；`internal/domain/models/vo/market_category_vo.go:11` | `symbol_analysis_application_test.go:198`（strict mock 未預期 `Create`）；`analysis_event_controller_test.go:101` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-3 | 找不到的標的不建立分析事件：證交所無 9999 → 發起被拒，告知「找不到此標的」，且沒有建立任何分析事件 | 發起被拒並顯示「找不到此標的」；不產生任何分析事件 | `symbol_resolution_service.go:31-37`；`internal/domain/service/news_search_errors.go:12` | `symbol_analysis_application_test.go:200-202`（twStock 9999）；`analysis_event_controller_test.go:105-107`（訊息與 404） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-4 | 未提供標的不建立分析事件 → 發起被拒，告知「標的為必填」 | 發起被拒並顯示「標的為必填」 | `symbol_resolution_service.go:25-28`；`internal/domain/models/vo/symbol_vo.go:8` | `symbol_analysis_application_test.go:199`；`analysis_event_controller_test.go:102` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-5 | 停用中的 API key 不能發起 → 發起被拒，告知「API key 尚未啟用」 | 以停用中的 key 發起分析被拒，顯示「API key 尚未啟用」 | `cmd/server/dependencies.go:126-128`（路由掛在 `RequireActiveApiKey` 後）；`internal/controller/api_key_controller.go:71-81` | `analysis_event_controller_test.go:131-139`（只測「未帶 key」→401）；`cmd/server/dependencies_test.go`（未帶 key→401）；停用 key 只在 `internal/controller/tests/api_key_controller_test.go:204-213` 對假路由 `/protected` 測 | shallow | produces-oracle | 🟠 mis-asserted |
| AC-6 | 6 小時內已完成的分析被重用：5 時 59 分前完成 → 取得既有編號，沒有建立新事件 | 回傳既有分析事件編號；不產生新分析事件 | `internal/domain/models/domains/analysis_event_domain.go:37-47`；`symbol_analysis_service.go:45-52` | `symbol_analysis_application_test.go:107`（無 `Create` 預期）；`internal/domain/models/domains/tests/analysis_domain_test.go:30` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-7 | 超過 6 小時的分析不重用：6 時 1 分前完成 → 建立新事件 | 產生一筆新分析事件 | `analysis_event_domain.go:43`；`symbol_analysis_service.go:56-64` | `symbol_analysis_application_test.go:134`；`analysis_domain_test.go:32` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-8 | 分析中的事件被重用 → 取得該分析中事件編號，沒有建立新事件 | 回傳分析中事件的編號；不產生新分析事件 | `analysis_event_domain.go:39-40`；`internal/infrastructure/persistence/analysis_event_repository.go:43-58` | `symbol_analysis_application_test.go:108`；`analysis_event_controller_test.go:81-90` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-9 | 失敗的分析不重用：唯一事件 1 小時前失敗 → 建立新事件 | 產生一筆新分析事件 | `analysis_event_repository.go:48`（只撈 running/succeeded）；`analysis_event_domain.go:44-45` | `symbol_analysis_application_test.go:135`；`internal/infrastructure/persistence/tests/analysis_repository_test.go:30-48` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-10 | 不同市場類別不重用：BTC（加密貨幣）1 小時前完成，發起 BTC 美股 → 建立新事件 | 產生一筆新分析事件 | `analysis_event_repository.go:46-47` | `symbol_analysis_application_test.go:159-174`；`analysis_repository_test.go:41,46-47` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-11 | 查詢已完成的分析事件 → 狀態「已完成」，並看到標的、市場類別、評等、信心指數、時間範圍、理由、關鍵事件、風險因子、分析佐證與建立時間 | 顯示已完成，並帶出列舉的全部結果欄位 | `symbol_analysis_service.go:139-167`；`analysis_event_domain.go:61-96`；`internal/domain/models/dto/analysis_result_dto.go` | `analysis_event_controller_test.go:141-160`（完整 JSON）；`analysis_domain_test.go:59-78` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-12 | 查詢分析中的分析事件 → 狀態「分析中」，沒有分析結果 | 顯示分析中，無任何結果 | `symbol_analysis_service.go:159-161`；`internal/domain/models/dto/analysis_event_dto.go`（`result,omitempty`） | `symbol_analysis_application_test.go:421` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-13 | 查詢失敗的分析事件（AI 服務無法使用）→ 狀態「失敗」與失敗原因「AI 服務暫時無法使用」 | 顯示失敗與原因「AI 服務暫時無法使用」 | `analysis_event_domain.go:67`；`symbol_analysis_service.go:88-90` | `symbol_analysis_application_test.go:422`；`analysis_event_controller_test.go:162-171` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-14 | 查詢不存在的分析事件 999 → 查詢被拒，告知「找不到此分析事件」 | 查詢被拒並顯示「找不到此分析事件」 | `symbol_analysis_service.go:144-146`；`symbol_analysis_errors.go:15`；`analysis_event_controller.go:16-18` | `analysis_event_controller_test.go:173-185` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-15 | 可查詢其他 API key 發起的分析事件 → 看到該事件與分析結果 | 以自己的有效 key 查他人發起的已完成事件，可看到事件與結果 | `symbol_analysis_service.go:139-148`（不以 API key 過濾） | `analysis_event_controller_test.go:141-160`（事件 `ApiKeyID: 99`、呈示 key ID 12） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-16 | 不合法的評等改為中性：「超級看多」→「中性」 | 結果評等為中性 | `internal/domain/models/domains/analysis_conclusion_domain.go:50-53` | `analysis_domain_test.go:105` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-17 | 信心指數超過 100 改為 100：130 → 100 | 結果信心指數為 100 | `analysis_conclusion_domain.go:58` | `analysis_domain_test.go:106` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-18 | 信心指數低於 0 改為 0：-5 → 0 | 結果信心指數為 0 | `analysis_conclusion_domain.go:58` | `analysis_domain_test.go:107` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-19 | 不合法的時間範圍改為短期：「長期」→「短期」 | 結果時間範圍為短期 | `analysis_conclusion_domain.go:54-57` | `analysis_domain_test.go:108` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-20 | 不是這次取得的新聞不能當關鍵事件 → 該則不在結果中 | 連結不在本次新聞中的關鍵事件被剔除 | `analysis_conclusion_domain.go:64-69`；`analysis_evidence_domain.go:32-38` | `analysis_domain_test.go:129-141`（`made-up`）；`symbol_analysis_application_test.go:274,299`（`https://invented`） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-21 | 關鍵事件與風險因子最多 5 項：7 則有效關鍵事件、7 條風險因子 → 只保留前 5 則 / 前 5 條 | 結果恰為前 5 則關鍵事件與前 5 條風險因子（依原順序） | `analysis_conclusion_domain.go:63-76` | `analysis_domain_test.go:129-150` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-22 | 沒有取得任何新聞時評等中性、信心指數 0：AI 給看多 80 → 中性、0 | 結果評等中性、信心指數 0 | `analysis_conclusion_domain.go:59-62` | `analysis_domain_test.go:121-127`；`symbol_analysis_application_test.go:365-380` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-23 | 分析佐證記錄這次取得的所有新聞：3 則不同新聞 → 佐證含 3 則的標題、連結、發布時間、新聞來源名稱 | 佐證恰為這 3 則新聞，各帶標題、連結、發布時間、來源名稱 | `analysis_evidence_domain.go:17-30`；`symbol_analysis_service.go:121` | `symbol_analysis_application_test.go:250-310`（兩次搜尋共 4 則、以連結去重成 3 則，四欄位全比對） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-24 | AI 沒有提供理由 → 分析事件失敗，原因「AI 未提供完整分析」 | 事件失敗，原因「AI 未提供完整分析」 | `analysis_conclusion_domain.go:46-49`；`symbol_analysis_service.go:100-105` | `symbol_analysis_application_test.go:327-329`；`analysis_domain_test.go:152-157` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-25 | AI 服務無法使用 → 分析事件失敗，原因「AI 服務暫時無法使用」 | 事件失敗，原因「AI 服務暫時無法使用」 | `symbol_analysis_service.go:86-91`；`internal/infrastructure/claude/claude_analyst_proxy.go:87-89` | `symbol_analysis_application_test.go:321-323`；`internal/infrastructure/claude/tests/claude_analyst_proxy_test.go:139-145` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-26 | AI 用完 5 輪工具使用仍未給出結論 → 失敗，原因「AI 未在限制內完成分析」 | 第 5 輪後仍無結論即失敗，原因「AI 未在限制內完成分析」，不超過 5 輪 | `symbol_analysis_service.go:16,83-85,125-130` | `symbol_analysis_application_test.go:333-335`（斷言恰 5 輪與原因） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-27 | AI 基於安全政策拒絕回答 → 失敗，原因「AI 拒絕分析此標的」 | 事件失敗，原因「AI 拒絕分析此標的」 | `claude_analyst_proxy.go:92`；`symbol_analysis_service.go:92-95` | `claude_analyst_proxy_test.go:129-137`；`symbol_analysis_application_test.go:324-326` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-28 | 服務重新啟動中斷分析：分析中的事件在服務重新啟動後失敗，原因「服務重新啟動，分析中斷」 | 重啟後殘留分析中事件變成失敗，原因「服務重新啟動，分析中斷」 | `cmd/server/main.go:22-24`；`symbol_analysis_service.go:150-155`；`analysis_event_repository.go:64-68` | `symbol_analysis_application_test.go:452-467`；`analysis_repository_test.go:73-89`；**無測試涵蓋「啟動時呼叫」** | shallow | produces-oracle | 🟠 mis-asserted |
| BR-1 | 評等值域：強烈看多、看多、中性、看空、強烈看空 | 結果評等只會是這五值之一 | `analysis_conclusion_domain.go:14-18,31,50-53`；`claude_analyst_proxy.go:143` | `analysis_domain_test.go:104-105` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-2 | 時間範圍值域：短期（兩週內）、中期（三個月內） | 結果時間範圍只會是短期或中期 | `analysis_conclusion_domain.go:20-21,32,54-57` | `analysis_domain_test.go:104,108` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-3 | 信心指數：0–100 整數 | 結果信心指數恆為 0 到 100 的整數 | `analysis_conclusion_domain.go:58`；`internal/domain/models/entities/analysis_result.go:11`（int） | `analysis_domain_test.go:106-107` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-4 | 關鍵事件：最多 5 則，只能引用這次取得的新聞（以連結比對） | 關鍵事件 ≤ 5，且每則連結都屬於本次取得的新聞 | `analysis_conclusion_domain.go:63-69` | `analysis_domain_test.go:129-141` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-5 | 風險因子：最多 5 條，空白項目略過 | 風險因子 ≤ 5，且不含空白項 | `analysis_conclusion_domain.go:70-76` | `analysis_domain_test.go:143-150` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-6 | 分析佐證：這次取得的所有新聞，以連結去重 | 佐證 = 本次所有取得新聞，同連結只出現一次 | `analysis_evidence_domain.go:17-30` | `analysis_domain_test.go:159-170`；`symbol_analysis_application_test.go:301-305` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-7 | 重用以「標的 + 市場類別」為單位；6 小時以分析完成時間起算，往回剛好 6 小時（含）以內者重用 | 同標的同市場、完成時間距今 ≤ 6 小時（含恰 6 小時）者重用；超過則不重用 | `analysis_event_domain.go:15,41-43`；`analysis_event_repository.go:43-58` | `analysis_domain_test.go:30-32`（含恰 6h）；`analysis_repository_test.go:30-48` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-8 | 一次分析最多 5 輪工具使用 | 一次分析對 AI 的往返不超過 5 輪 | `symbol_analysis_service.go:16,85` | `symbol_analysis_application_test.go:333-335,358` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-9 | 分析事件記錄發起的 API key、開始與結束時間、使用的 AI 模型與 AI 用量（輸入、輸出量） | 每筆分析事件可看到發起 key、開始 / 結束時間、**實際使用**的 AI 模型、輸入與輸出用量合計 | `analysis_event_domain.go:26-35,102-108`；`symbol_analysis_service.go:56,87`；`claude_analyst_proxy.go:59-61,83-84,93` | `symbol_analysis_application_test.go:96,309`；`analysis_domain_test.go:43-57`；`analysis_repository_test.go:50-71` | shallow | diverges | 🔴 violation |
| BR-10 | Edge：AI 搜尋新聞時新聞來源全部失敗 → 告知 AI 這次沒有取得新聞，由 AI 繼續；最後若完全沒有新聞依「沒有取得任何新聞」規則處理 | 該次搜尋回報 AI「沒有取得新聞」，分析不中止、AI 可繼續；最終無新聞則中性 + 0 | `internal/domain/service/news_search_service.go:78-79`；`symbol_analysis_service.go:116-120`；`analysis_conclusion_domain.go:59-62` | 「無新聞 → 中性 0」有 `symbol_analysis_application_test.go:365-380`；**無測試讓新聞來源全部失敗** | no-test | produces-oracle | 🟡 partial |
| BR-11 | Edge：AI 搜尋的相關標的找不到 → 告知 AI 找不到，由 AI 繼續 | 該次搜尋回報 AI「找不到此標的」，分析不中止、AI 可繼續 | `symbol_resolution_service.go:35-37,44-46`；`symbol_analysis_service.go:116-120` | **無測試**：`symbol_analysis_application_test.go:269,296` 只測「市場類別不合法」錯誤分支 | no-test | produces-oracle | 🟡 partial |
| BR-12 | Edge：保存分析結果失敗 → 分析事件失敗，原因「分析結果保存失敗」 | 事件失敗，原因「分析結果保存失敗」 | `symbol_analysis_service.go:107-110`；`symbol_analysis_errors.go:10` | `symbol_analysis_application_test.go:336-340` | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-1 | Performance：發起分析在標的辨識完成後立即回應，不等待 AI | 發起回應不需等 AI 完成即回傳 | `internal/application/symbol_analysis_application.go:24-27`（`go` + `context.WithoutCancel`） | `symbol_analysis_application_test.go:67-99`、`analysis_event_controller_test.go:57-79`：AI mock 立即返回、channel 有緩衝，若改為同步執行仍會通過 | shallow | produces-oracle | 🟠 mis-asserted |
| NFR-2 | Security：沿用 API key 把關；AI 只能使用「搜尋標的新聞」與「提交結論」兩項能力 | 分析端點需有效 key；AI 可用能力恰為這兩項 | `cmd/server/dependencies.go:126-129`；`claude_analyst_proxy.go:123-154`；`symbol_analysis_service.go:114-124` | `cmd/server/dependencies_test.go`（分析路由未帶 key→401）；`claude_analyst_proxy_test.go:60-68`（tools 恰兩個、strict） | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-3 | Cost：重用規則避免重複分析；AI 用量逐筆記錄 | 可重用時不再呼叫 AI；每筆事件記錄用量 | `symbol_analysis_service.go:49-52`；`symbol_analysis_application.go:21-23`；`analysis_event_domain.go:106-107` | `symbol_analysis_application_test.go:101-127`（重用時 analyst mock 未預期呼叫）、`:309`（300/30） | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-4 | Analytics / Tracking：分析事件記錄 AI 用量 | 每筆分析事件可看到 AI 輸入 / 輸出用量 | `analysis_event_domain.go:106-107`；`internal/domain/models/entities/analysis_event.go:13-14` | `symbol_analysis_application_test.go:309`；`analysis_repository_test.go:58-67` | asserts-oracle | produces-oracle | ✅ conforms |

### 非符合條款說明

- **BR-9（🔴 violation）**：分析事件的 `Model` 在建立時就寫入設定值（`symbol_analysis_service.go:56` 取 `analystProxy.ModelName()`，即 `claude_analyst_proxy.go:59-61` 回傳的設定模型），之後從不更新。但 Proxy 自己開啟了 server-side fallback（`claude_analyst_proxy.go:83-84`，`fallbacks: "default"`）：所請求的模型拒答時，伺服器會改由其他模型產生這一輪內容（SDK 說明：是否由 fallback 模型回應，要看 `usage.iterations` 有沒有 `fallback_message`）。這時被保存的結論來自 fallback 模型，事件卻記成設定的模型，不符合「使用的 AI 模型」。Proxy 沒有讀 `response.Model` 或 `usage.iterations`，`AnalystTurnVo` 也沒有欄位能把實際模型帶回 domain。一般路徑（沒有觸發 fallback）符合 oracle。測試（`symbol_analysis_application_test.go:309`）只斷言設定的模型名稱，fallback 路徑下會照樣通過，所以判為 shallow。附帶注意：用量只加總 `usage.input_tokens`/`output_tokens`（`claude_analyst_proxy.go:93`），沒有計入 `cache_read_input_tokens`/`cache_creation_input_tokens`。目前快取前綴（tools + system）低於最小可快取長度，實務上應為 0；但只要日後加長 prompt，輸入用量就會少算。
- **AC-5（🟠）**：停用 key 被拒的行為只對測試用假路由 `/protected` 驗證過（`api_key_controller_test.go:204-213`）。分析路由的測試只驗證「未帶 key → 401」（`analysis_event_controller_test.go:131-139`、`dependencies_test.go`），假如分析路由改用只檢查「是否帶 key」的關卡，這些測試仍會通過。前一切片的新聞路由則有 `news_controller_test.go:82` 對停用 key 斷言「API key 尚未啟用」，可比照補上。
- **AC-28（🟠）**：`FailInterruptedAnalysisEvents` 與 `FailAllRunning` 各自有測試，但觸發點「服務啟動時呼叫」（`cmd/server/main.go:22-24`）沒有任何測試覆蓋；刪掉這一行，全部測試仍會綠燈。
- **NFR-1（🟠）**：`TestStartSymbolAnalysis_CreatesARunningAnalysisAndAnalyzesInTheBackground` 用的 `analyzed` channel 有緩衝（`symbol_analysis_application_test.go:77`），AI mock 也立即返回；若把 `symbol_analysis_application.go:26` 的 `go` 拿掉改成同步執行，測試仍會通過。需要一個讓 AI 呼叫阻塞、同時斷言發起已先回應的測試。
- **BR-10 / BR-11（🟡）**：程式碼把所有搜尋錯誤一律轉成 `is_error` 工具結果，AI 可以繼續（`symbol_analysis_service.go:116-120`），內容分別是「新聞來源暫時無法使用」與「找不到此標的」。但測試只涵蓋「市場類別不合法」這一種錯誤分支（`symbol_analysis_application_test.go:269,296`），「新聞來源全部失敗」與「相關標的找不到」兩條 oracle 都沒有測試斷言。

## Orphans（無對應條款的行為）

| Code | Description | Verdict |
|------|-------------|---------|
| `internal/controller/analysis_event_controller.go:35-37` | JSON body 格式錯誤時回 400 `invalid_request_body`「請求內容格式錯誤」；空 body（`io.EOF`）則當作空欄位，最後落到「市場類別只能是…」 | undocumented |
| `internal/domain/service/symbol_resolution_service.go:31-33,40-42` + `analysis_event_controller.go:16` | 發起分析時，標的目錄（TWSE / CoinGecko）查詢失敗會回 502 `news_providers_unavailable`「新聞來源暫時無法使用」。PRD 沒有這個情境，且訊息把「標的目錄」講成「新聞來源」，對發起分析的使用者有誤導 | undocumented |
| `internal/controller/error_response_table.go:25`；`symbol_analysis_service.go:47,62,73,142,164` | 資料存取失敗回 503 `service_unavailable`「服務暫時無法使用」（ARCH 有記載，PRD 沒有） | undocumented（ARCH 已記載） |
| `symbol_analysis_service.go:53-67` | 併發發起時 `Create` 撞上唯一索引，重查也找不到可重用事件（對方已失敗），回 503。PRD 沒有定義 | undocumented |
| `analysis_event_controller.go:48-51` | 新建回 202、重用回 200（ARCH 有記載，PRD 沒有區分） | undocumented（ARCH 已記載） |
| `symbol_analysis_service.go:50` | 重用已完成的事件時，發起回應會直接附上完整分析結果 | undocumented |
| `symbol_analysis_service.go:96-98` | AI 該輪既沒有工具呼叫也沒有結論（純文字結束）時視為失敗，原因「AI 未提供完整分析」（ARCH 有記載；PRD 只定義「結論沒有理由」） | undocumented（ARCH 已記載） |
| `symbol_analysis_service.go:85,114-125` | 第 5 輪 AI 若要求搜尋，仍會實際執行新聞搜尋並寫入佐證，但結果永遠不會交回 AI，隨即以「未在限制內完成」失敗，白做外部呼叫 | undocumented |
| `internal/domain/models/domains/analysis_conclusion_domain.go:64-69` | AI 重複引用同一連結時，關鍵事件會出現重複項（佐證有以連結去重，關鍵事件沒有），會佔用 5 則上限 | undocumented |
| `analysis_event_controller.go:56-59` | 非正整數的 id（`abc`、`-1`）視同不存在，回 404「找不到此分析事件」（ARCH 有記載） | undocumented（ARCH 已記載） |
| `internal/infrastructure/claude/claude_analyst_proxy.go:83-84` | 開啟 server-side fallback：請求的模型拒答時改由其他模型作答，因此「AI 拒絕分析此標的」只在所有模型都拒答時才會出現。PRD 沒有提到（並導致 BR-9 違反） | undocumented |
| `cmd/server/config.go`（`AI_ANALYSIS_MODEL` / `AI_ANALYSIS_EFFORT`） | 模型與 effort 可由環境變數設定；不合法的 effort 會靜默改用 `high`（ARCH 有記載） | undocumented（ARCH 已記載） |

Out of Scope 檢查：沒有發現分析時價格 / 回測、定時自動分析、分析事件列表或依標的查歷史、取消分析的實作（路由只有 `POST /analysis-events` 與 `GET /analysis-events/:analysisEventId`，見 `cmd/server/dependencies.go:128-129`）。**沒有超出範圍的違規。**

## Summary

- Conforms: 38/44 clauses ✅（86%）
- Violations: BR-9（實際使用的 AI 模型在 server-side fallback 時記錄錯誤）
- Mis-asserted: AC-5、AC-28、NFR-1（綠燈測試只斷言較弱的條件）
- Partial: BR-10、BR-11（沒有測試斷言 oracle）
- Gaps: 無
- Unclear: 無
- Orphans: 12（皆為 undocumented，沒有 Out of Scope 違規）

---

## 稽核後修正紀錄（2026-10-06）

| 項目 | 處理 |
| :--- | :--- |
| BR-9 fallback 時記錄錯誤模型 | 修正：Proxy 回傳實際作答模型（`response.Model`），domain 以 `RecordAnsweringModel` 寫入；用量的輸入量含快取寫入 / 讀取。補測試 |
| AC-5 停用 key 未在分析路由測試 | 補測試：停用 key 呼叫發起分析 → 403「API key 尚未啟用」 |
| AC-28 啟動中斷分析未被測試 | 啟動流程抽成 `prepareRouter`，補測試證明啟動時先執行中斷處理、失敗則不啟動 |
| NFR-1 測試未證明非同步 | 補測試：AI 呼叫卡住時發起已回應 |
| BR-10、BR-11 缺測試 | 補測試：新聞來源全失敗、相關標的找不到 → 錯誤工具結果回給 AI，分析繼續並完成 |
| O-8 第 5 輪白抓新聞 | 修正：最後一輪不再搜尋，PRD 補規則 |
| O-9 關鍵事件重複 | 修正：以連結去重，PRD 補規則 |
| O-11 fallback 語意 | PRD 補規則 |
| O-6 重用已完成事件附結果 | PRD 補規則 |
| O-2 標的清單失敗訊息 | 保留：沿用 symbol-news-search PRD 的既有規則（辨識資料取不到 → 新聞來源暫時無法使用） |
| O-1、O-3、O-4、O-5、O-7、O-10、O-12 | 保留，ARCH 已記載 |
