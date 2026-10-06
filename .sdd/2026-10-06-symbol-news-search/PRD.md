# Product Requirements Document (PRD) — 標的新聞搜尋

**Status:** Finalized
**Version:** v1.0
**Owner:** James Hsueh
**Stakeholders:** Engineering

---

## 1. Background & Goal (Why & Goal)

- **Problem Statement:** 標的的資訊面散落在不同新聞來源，且不同市場適合的來源不同。AI 分析之前，需要一個依「標的 + 市場類別」自動選擇新聞來源、合併並整理近期新聞的能力。
- **Expected Outcome:**
  - 提供標的與市場類別即可取得最近 7 天、去重、由新到舊、最多 30 則的新聞。
  - 單一新聞來源故障不會讓整次搜尋失敗。
  - 這個能力之後可直接作為 AI 分析時使用的「爬取新聞」工具。
- **Out of Scope:**
  - AI 分析。
  - 新聞內文全文擷取；只使用新聞來源提供的標題與摘要。
  - 新聞保存與快取。
  - 上櫃、興櫃股票。
  - 使用者自訂時間範圍、數量或新聞來源。

---

## 2. User Personas

- **Primary Role(s):** 持有已啟用 API key 的使用者（外部開發者）。
- **Usage Context:** 以程式呼叫，通常在做投資研究前查詢某個標的的近期新聞；無圖形介面。

---

## 3. User Stories & Acceptance Criteria

### US-01 — 只有已啟用的 API key 能搜尋新聞 [priority: P0]
**As a** administrator, **I want** 新聞搜尋只開放給已啟用的 API key, **so that** 只有我核准的使用者能使用服務。

```gherkin
Scenario: 已啟用的 API key 可以搜尋
  Given 使用者持有已啟用的 API key
  When 使用者搜尋標的 BTC、市場類別加密貨幣的新聞
  Then 使用者取得 BTC 的新聞清單

Scenario: 停用中的 API key 被拒
  Given 使用者持有停用中的 API key
  When 使用者搜尋新聞
  Then 搜尋被拒，告知「API key 尚未啟用」

Scenario: 未提供 API key 被拒
  Given 使用者沒有提供 API key
  When 使用者搜尋新聞
  Then 搜尋被拒，告知「需要提供 API key」
```

### US-02 — 市場類別與標的必須有效 [priority: P0]
**As a** 使用者, **I want** 輸入錯誤時得到明確原因, **so that** 我能修正後重試。

```gherkin
Scenario: 不支援的市場類別被拒
  Given 使用者提供市場類別「港股」
  When 使用者搜尋新聞
  Then 搜尋被拒，告知「市場類別只能是 crypto、twStock、usStock」

Scenario: 未提供市場類別被拒
  Given 使用者沒有提供市場類別
  When 使用者搜尋新聞
  Then 搜尋被拒，告知「市場類別只能是 crypto、twStock、usStock」

Scenario: 未提供標的被拒
  Given 使用者沒有提供標的，或標的只有空白
  When 使用者搜尋新聞
  Then 搜尋被拒，告知「標的為必填」

Scenario: 標的去除空白並轉為大寫
  Given 使用者提供標的「 aapl 」、市場類別美股
  When 使用者搜尋新聞
  Then 系統以 AAPL 搜尋
  And 結果中的標的顯示為 AAPL

Scenario: 不存在的台股標的被拒
  Given 證交所上市公司中沒有代號 9999
  When 使用者搜尋標的 9999、市場類別台股的新聞
  Then 搜尋被拒，告知「找不到此標的」

Scenario: 無法辨識的加密貨幣被拒
  Given 沒有任何幣種的代號是 NOTACOIN
  When 使用者搜尋標的 NOTACOIN、市場類別加密貨幣的新聞
  Then 搜尋被拒，告知「找不到此標的」

Scenario: 查無新聞的美股回傳空清單
  Given 所有美股新聞來源都沒有 ZZZZ 的新聞
  When 使用者搜尋標的 ZZZZ、市場類別美股的新聞
  Then 搜尋成功，新聞清單為空
```

### US-03 — 依市場類別選擇新聞來源 [priority: P0]
**As a** 使用者, **I want** 系統替我挑選適合該市場的新聞來源與搜尋字, **so that** 我不需要了解各來源的差異。

```gherkin
Scenario: 台股以公司簡稱向鉅亨網與 Google 新聞（繁中）搜尋
  Given 證交所上市公司 2330 的公司簡稱為「台積電」
  When 使用者搜尋標的 2330、市場類別台股的新聞
  Then 系統以「台積電」向鉅亨網與 Google 新聞（繁體中文）取得新聞

Scenario: 美股以代號向 Yahoo 財經與 Google 新聞（英文）搜尋
  When 使用者搜尋標的 AAPL、市場類別美股的新聞
  Then 系統以 AAPL 向 Yahoo 財經與 Google 新聞（英文）取得新聞

Scenario: 加密貨幣以幣種名稱與代號篩選 CoinDesk、Cointelegraph，並向 Google 新聞（英文）搜尋
  Given 代號 BTC 對應的幣種名稱為 Bitcoin
  When 使用者搜尋標的 BTC、市場類別加密貨幣的新聞
  Then 系統以 Bitcoin 向 Google 新聞（英文）搜尋
  And CoinDesk 與 Cointelegraph 只保留標題或摘要提到「Bitcoin」或「BTC」的新聞
```

### US-04 — 新聞清單整理 [priority: P0]
**As a** 使用者, **I want** 拿到乾淨、近期、排序好的新聞, **so that** 我能直接閱讀或交給分析使用。

```gherkin
Scenario: 只保留最近 7 天內的新聞
  Given 一則新聞發布於 6 天 23 小時前
  And 一則新聞發布於 7 天 1 小時前
  When 使用者搜尋新聞
  Then 結果只包含發布於 6 天 23 小時前的那則

Scenario: 相同標題的新聞只保留一則
  Given 鉅亨網與 Google 新聞都有標題為「台積電法說會」的新聞
  When 使用者搜尋新聞
  Then 結果中標題為「台積電法說會」的新聞只有一則

Scenario: 依發布時間由新到舊排序
  Given 三則新聞分別發布於前天、今天、昨天
  When 使用者搜尋新聞
  Then 結果順序為今天、昨天、前天

Scenario: 最多回傳 30 則
  Given 符合條件的新聞有 45 則
  When 使用者搜尋新聞
  Then 結果只包含最新的 30 則

Scenario: 每則新聞包含完整資訊
  Given 鉅亨網有一則附摘要的新聞
  When 使用者搜尋新聞
  Then 該則新聞包含標題、連結、發布時間、新聞來源名稱「鉅亨網」與摘要
```

### US-05 — 新聞來源故障 [priority: P0]
**As a** 使用者, **I want** 部分新聞來源故障時仍拿到可用結果, **so that** 單一來源不穩不會讓我完全拿不到新聞。

```gherkin
Scenario: 單一新聞來源失敗仍回傳其他結果
  Given 搜尋台股時 Google 新聞沒有回應
  And 鉅亨網正常回傳新聞
  When 使用者搜尋新聞
  Then 搜尋成功，結果包含鉅亨網的新聞
  And 告知 Google 新聞這次沒有取得資料

Scenario: 所有新聞來源都失敗
  Given 該市場類別的所有新聞來源都沒有回應
  When 使用者搜尋新聞
  Then 搜尋被拒，告知「新聞來源暫時無法使用」

Scenario: 辨識標的所需的資料取不到
  Given 證交所上市公司清單暫時取不到
  When 使用者搜尋台股新聞
  Then 搜尋被拒，告知「新聞來源暫時無法使用」
```

---

## 4. Business Flow & Logic

- **Flow:** 驗證 API key → 驗證市場類別與標的 → 辨識標的（台股查公司簡稱、加密貨幣查幣種名稱、美股直接使用代號）→ 同時向該市場類別的所有新聞來源取新聞 → 篩選（加密貨幣專門來源依名稱 / 代號）→ 只留 7 天內 → 以標題去重 → 由新到舊排序 → 取前 30 則。
- **Core Business Rules:**
  - 市場類別與新聞來源對應：台股 → 鉅亨網、Google 新聞（繁中）；美股 → Yahoo 財經、Google 新聞（英文）；加密貨幣 → CoinDesk、Cointelegraph、Google 新聞（英文）。
  - 「同一則新聞」：標題去除前後空白、不分大小寫後相同；保留先出現（較新）的那則。
  - 7 天以「搜尋當下」往回推算，發布時間剛好 7 天前（含）以內的新聞保留。
  - 加密貨幣代號對應多個幣種時，取市值排名最前者。
  - 證交所上市公司清單與幣種辨識結果保留 24 小時後重新取得。
- **Edge Cases:**
  - 單一新聞來源超過 10 秒未回應視為失敗。
  - 新聞來源回傳無法辨識的發布時間 → 該則新聞略過。
  - 新聞來源回傳的資料格式無法解析，或不是預期的新聞資料（例如錯誤訊息）→ 視為該新聞來源失敗。

---

## 5. UI/UX Design & Interaction

N/A — 無圖形介面。

---

## 6. Non-Functional Requirements

- **Performance:** 標的辨識完成後才能決定搜尋字，因此依序進行兩段：標的辨識最多等待 10 秒（結果保留 24 小時，命中時不需等待）；各新聞來源同時查詢，最多等待 10 秒。同時進行的搜尋不會因其他標的的辨識較慢而排隊等待。
- **Security:** 沿用 API key 把關。
- **Compatibility:** N/A
- **Analytics / Tracking:** N/A

---

## 7. Dependencies & Risks

- **External Dependencies:** 鉅亨網、Google 新聞、Yahoo 財經、CoinDesk、Cointelegraph、證交所上市公司開放資料、CoinGecko 幣種搜尋。
- **Known Risks:**
  - Google 新聞的使用條款限個人非商業用途，對外提供服務有法律風險。
  - 鉅亨網、Yahoo 財經的資料取得方式非正式公開服務，格式可能無預警變更。
  - CoinGecko 免費額度有頻率限制。

---

## 8. Appendix

- `.sdd/2026-10-06-symbol-news-search/BRIEF.md`
- `.sdd/2026-10-06-api-key-management/PRD.md`（API key 把關規則）
