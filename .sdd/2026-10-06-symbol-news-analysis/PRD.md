# Product Requirements Document (PRD) — 標的資訊面分析

**Status:** Finalized
**Version:** v1.0
**Owner:** James Hsueh
**Stakeholders:** Engineering

---

## 1. Background & Goal (Why & Goal)

- **Problem Statement:** 使用者想快速知道某個標的近期的資訊面偏多還是偏空、為什麼、根據是什麼。逐則閱讀新聞耗時，且容易只看到單方面說法。
- **Expected Outcome:**
  - 提供標的與市場類別即可發起分析，立即拿到分析事件編號；之後以編號取得標準化分析結果。
  - 每個分析結果都可追溯到 AI 實際讀過的新聞（分析佐證）。
  - 同一標的 6 小時內重複發起不會重複消耗 AI 費用。
- **Out of Scope:**
  - 分析時價格與回測。
  - 定時自動分析。
  - 分析事件列表、依標的查詢歷史分析。
  - 取消分析。

---

## 2. User Personas

- **Primary Role(s):** 持有已啟用 API key 的使用者（外部開發者）。
- **Usage Context:** 以程式呼叫；發起後輪詢查詢分析事件，通常數十秒內完成。

---

## 3. User Stories & Acceptance Criteria

### US-01 — 發起分析 [priority: P0]
**As a** 使用者, **I want** 提供標的與市場類別就發起一次資訊面分析, **so that** 我不必自己讀完所有新聞。

```gherkin
Scenario: 發起分析成功
  Given 使用者持有已啟用的 API key
  And BTC（加密貨幣）在近 6 小時內沒有分析事件
  When 使用者發起 BTC、市場類別加密貨幣的分析
  Then 使用者取得一個新的分析事件編號
  And 該分析事件的狀態為「分析中」

Scenario: 不支援的市場類別不建立分析事件
  Given 使用者提供市場類別「港股」
  When 使用者發起分析
  Then 發起被拒，告知「市場類別只能是 crypto、twStock、usStock」
  And 沒有建立任何分析事件

Scenario: 找不到的標的不建立分析事件
  Given 證交所上市公司中沒有代號 9999
  When 使用者發起標的 9999、市場類別台股的分析
  Then 發起被拒，告知「找不到此標的」
  And 沒有建立任何分析事件

Scenario: 未提供標的不建立分析事件
  Given 使用者沒有提供標的
  When 使用者發起分析
  Then 發起被拒，告知「標的為必填」

Scenario: 停用中的 API key 不能發起
  Given 使用者持有停用中的 API key
  When 使用者發起分析
  Then 發起被拒，告知「API key 尚未啟用」
```

### US-02 — 重用既有分析 [priority: P0]
**As a** administrator, **I want** 同一標的短時間內不重複分析, **so that** AI 費用不會被重複消耗。

```gherkin
Scenario: 6 小時內已完成的分析被重用
  Given BTC（加密貨幣）有一筆 5 小時 59 分前完成的分析事件
  When 使用者發起 BTC、市場類別加密貨幣的分析
  Then 使用者取得那筆既有分析事件的編號
  And 沒有建立新的分析事件

Scenario: 超過 6 小時的分析不重用
  Given BTC（加密貨幣）最近一筆完成的分析事件在 6 小時 1 分前完成
  When 使用者發起 BTC、市場類別加密貨幣的分析
  Then 建立一筆新的分析事件

Scenario: 分析中的事件被重用
  Given BTC（加密貨幣）有一筆分析中的分析事件
  When 使用者發起 BTC、市場類別加密貨幣的分析
  Then 使用者取得那筆分析中分析事件的編號
  And 沒有建立新的分析事件

Scenario: 失敗的分析不重用
  Given BTC（加密貨幣）唯一的分析事件在 1 小時前失敗
  When 使用者發起 BTC、市場類別加密貨幣的分析
  Then 建立一筆新的分析事件

Scenario: 不同市場類別不重用
  Given BTC（加密貨幣）有一筆 1 小時前完成的分析事件
  When 使用者發起 BTC、市場類別美股的分析
  Then 建立一筆新的分析事件
```

### US-03 — 查詢分析事件 [priority: P0]
**As a** 使用者, **I want** 以分析事件編號查詢進度與結果, **so that** 分析完成後我能取得結論。

```gherkin
Scenario: 查詢已完成的分析事件
  Given 一筆已完成的分析事件
  When 使用者查詢該分析事件
  Then 看到狀態「已完成」
  And 看到分析結果：標的、市場類別、評等、信心指數、時間範圍、理由、關鍵事件、風險因子、分析佐證與建立時間

Scenario: 查詢分析中的分析事件
  Given 一筆分析中的分析事件
  When 使用者查詢該分析事件
  Then 看到狀態「分析中」
  And 沒有分析結果

Scenario: 查詢失敗的分析事件
  Given 一筆因 AI 服務無法使用而失敗的分析事件
  When 使用者查詢該分析事件
  Then 看到狀態「失敗」與失敗原因「AI 服務暫時無法使用」

Scenario: 查詢不存在的分析事件
  Given 沒有編號為 999 的分析事件
  When 使用者查詢分析事件 999
  Then 查詢被拒，告知「找不到此分析事件」

Scenario: 可查詢其他 API key 發起的分析事件
  Given 一筆由另一把 API key 發起並已完成的分析事件
  When 使用者以自己已啟用的 API key 查詢
  Then 看到該分析事件與分析結果
```

### US-04 — 分析結果標準化 [priority: P0]
**As a** 使用者, **I want** 分析結果永遠是固定的格式與合理的值, **so that** 我的程式可以直接使用。

```gherkin
Scenario: 不合法的評等改為中性
  Given AI 給出的評等為「超級看多」
  When 分析完成
  Then 分析結果的評等為「中性」

Scenario: 信心指數超過 100 改為 100
  Given AI 給出的信心指數為 130
  When 分析完成
  Then 分析結果的信心指數為 100

Scenario: 信心指數低於 0 改為 0
  Given AI 給出的信心指數為 -5
  When 分析完成
  Then 分析結果的信心指數為 0

Scenario: 不合法的時間範圍改為短期
  Given AI 給出的時間範圍為「長期」
  When 分析完成
  Then 分析結果的時間範圍為「短期」

Scenario: 不是這次取得的新聞不能當關鍵事件
  Given AI 列出的一則關鍵事件連結不在這次取得的新聞中
  When 分析完成
  Then 該則關鍵事件不在分析結果中

Scenario: 關鍵事件與風險因子最多 5 項
  Given AI 列出 7 則有效關鍵事件與 7 條風險因子
  When 分析完成
  Then 分析結果只保留前 5 則關鍵事件與前 5 條風險因子

Scenario: 沒有取得任何新聞時評等中性、信心指數 0
  Given 這次分析沒有取得任何新聞
  And AI 給出評等看多、信心指數 80
  When 分析完成
  Then 分析結果的評等為「中性」、信心指數為 0

Scenario: 分析佐證記錄這次取得的所有新聞
  Given AI 這次透過搜尋取得 3 則不同的新聞
  When 分析完成
  Then 分析佐證包含這 3 則新聞的標題、連結、發布時間與新聞來源名稱
```

### US-05 — 分析失敗 [priority: P0]
**As a** 使用者, **I want** 分析失敗時知道原因, **so that** 我能決定是否重新發起。

```gherkin
Scenario: AI 沒有提供理由
  Given AI 的結論沒有理由
  When 分析結束
  Then 分析事件失敗，原因「AI 未提供完整分析」

Scenario: AI 服務無法使用
  Given AI 服務沒有回應
  When 分析進行中
  Then 分析事件失敗，原因「AI 服務暫時無法使用」

Scenario: AI 在限制內未給出結論
  Given AI 用完 5 輪工具使用仍未給出結論
  When 分析進行中
  Then 分析事件失敗，原因「AI 未在限制內完成分析」

Scenario: AI 拒絕分析
  Given AI 基於安全政策拒絕回答
  When 分析進行中
  Then 分析事件失敗，原因「AI 拒絕分析此標的」

Scenario: 服務重新啟動中斷分析
  Given 一筆分析中的分析事件
  When 服務重新啟動
  Then 該分析事件失敗，原因「服務重新啟動，分析中斷」
```

---

## 4. Business Flow & Logic

- **Flow:** 驗證 API key → 驗證市場類別、標的並辨識標的 → 尋找可重用的分析事件（分析中，或 6 小時內已完成）→ 有則回傳 → 否則建立分析事件（分析中）並回傳 → 背景：AI 以「搜尋標的新聞」取得新聞、可搜尋相關標的 → AI 提交結論 → 正規化 → 保存分析結果、分析事件改為已完成；任何失敗 → 分析事件改為失敗並記錄原因。
- **Core Business Rules:**
  - 評等值域：強烈看多、看多、中性、看空、強烈看空。
  - 時間範圍值域：短期（兩週內）、中期（三個月內）。
  - 信心指數：0–100 整數。
  - 關鍵事件：最多 5 則，只能引用這次取得的新聞（以連結比對），同一則新聞只列一次。風險因子：最多 5 條，空白項目略過。
  - 分析佐證：這次取得的所有新聞，以連結去重。
  - 重用以「標的 + 市場類別」為單位；6 小時以分析完成時間起算，往回剛好 6 小時（含）以內者重用。
  - 一次分析最多 5 輪工具使用；第 5 輪 AI 若仍要求搜尋，不再實際搜尋（AI 已無機會閱讀），直接以「AI 未在限制內完成分析」結束。
  - 「使用的 AI 模型」記錄實際作答的模型：設定的模型拒答時，AI 服務會改由其他模型作答，此時記錄後者；只有所有模型都拒答才以「AI 拒絕分析此標的」結束。
  - 重用已完成的分析事件時，發起的回應直接附上分析結果。
  - 一次分析最多進行 10 分鐘，逾時以「分析逾時」結束。分析中超過 15 分鐘的分析事件視為卡住：不再重用，下一次發起時改為失敗（「分析逾時」）並重新分析。
  - 同時進行的分析有上限（預設 4 個）。額滿時仍可重用既有分析；需要新分析則拒絕，告知「目前分析數量已達上限，請稍後再試」。
  - 分析事件記錄發起的 API key、開始與結束時間、使用的 AI 模型與 AI 用量（輸入、輸出量）。
- **Edge Cases:**
  - AI 搜尋新聞時新聞來源全部失敗 → 告知 AI 這次沒有取得新聞，由 AI 繼續；最後若完全沒有新聞則依「沒有取得任何新聞」規則處理。
  - AI 搜尋的相關標的找不到 → 告知 AI 找不到，由 AI 繼續。
  - 保存分析結果失敗 → 分析事件失敗，原因「分析結果保存失敗」。

---

## 5. UI/UX Design & Interaction

N/A — 無圖形介面。

---

## 6. Non-Functional Requirements

- **Performance:** 發起分析在標的辨識完成後立即回應，不等待 AI。
- **Security:** 沿用 API key 把關；AI 只能使用「搜尋標的新聞」與「提交結論」兩項能力。
- **Cost:** 重用規則避免重複分析；AI 用量逐筆記錄。
- **Analytics / Tracking:** 分析事件記錄 AI 用量。

---

## 7. Dependencies & Risks

- **External Dependencies:** Anthropic Claude（AI 分析）、標的新聞搜尋（前一切片）。
- **Known Risks:**
  - AI 結論非投資建議；評等可能出錯，需靠下一切片的回測驗證。
  - AI 服務費用與頻率限制。

---

## 8. Appendix

- `.sdd/2026-10-06-symbol-news-analysis/BRIEF.md`
- `.sdd/2026-10-06-symbol-news-search/PRD.md`
