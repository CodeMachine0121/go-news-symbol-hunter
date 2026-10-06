# Product Requirements Document (PRD) — 上櫃股票支援

**Status:** Finalized
**Version:** v1.0
**Owner:** James Hsueh
**Stakeholders:** Engineering

---

## 1. Background & Goal (Why & Goal)

- **Problem Statement:** 台股目前只辨識證交所上市股票，上櫃股票（例如 6182 合晶）在新聞搜尋與分析時都被告知找不到此標的。
- **Expected Outcome:**
  - 上櫃股票可以搜尋新聞、發起分析，並記錄分析時價格。
  - 上市股票的行為完全不變。
- **Out of Scope:**
  - 興櫃股票。
  - 上櫃股票的即時價格。
  - 新增新聞來源。

---

## 2. User Personas

- **Primary Role(s):** 持有已啟用 API key 的使用者。
- **Usage Context:** 以台股市場類別查詢上櫃股票代號。

---

## 3. User Stories & Acceptance Criteria

### US-01 — 台股同時辨識上市與上櫃股票 [priority: P0]
**As a** 使用者, **I want** 用上櫃股票代號搜尋新聞與發起分析, **so that** 我關注的上櫃公司也能被分析。

```gherkin
Scenario: 上市股票行為不變
  Given 證交所上市公司 2330 的公司簡稱為「台積電」
  When 使用者搜尋標的 2330、市場類別台股的新聞
  Then 系統以「台積電」搜尋新聞

Scenario: 上櫃股票以櫃買中心簡稱搜尋
  Given 證交所上市公司中沒有 6182
  And 櫃買中心資料中 6182 的公司簡稱為「合晶」
  When 使用者搜尋標的 6182、市場類別台股的新聞
  Then 系統以「合晶」搜尋新聞

Scenario: 上市上櫃都找不到
  Given 證交所與櫃買中心都沒有 9999
  When 使用者搜尋標的 9999、市場類別台股的新聞
  Then 搜尋被拒，告知「找不到此標的」

Scenario: 證交所取不到但櫃買中心找到
  Given 證交所上市公司清單暫時取不到
  And 櫃買中心資料中 6182 的公司簡稱為「合晶」
  When 使用者搜尋標的 6182、市場類別台股的新聞
  Then 系統以「合晶」搜尋新聞

Scenario: 證交所取不到且櫃買中心沒有
  Given 證交所上市公司清單暫時取不到
  And 櫃買中心資料中沒有 2330
  When 使用者搜尋標的 2330、市場類別台股的新聞
  Then 搜尋被拒，告知「新聞來源暫時無法使用」

Scenario: 證交所沒有且櫃買中心取不到
  Given 證交所上市公司中沒有 6182
  And 櫃買中心資料暫時取不到
  When 使用者搜尋標的 6182、市場類別台股的新聞
  Then 搜尋被拒，告知「新聞來源暫時無法使用」

Scenario: 上櫃股票可以發起分析
  Given 櫃買中心資料中 6182 的公司簡稱為「合晶」
  When 使用者發起標的 6182、市場類別台股的分析
  Then 使用者取得分析事件編號，狀態「分析中」
```

### US-02 — 上櫃股票的分析時價格 [priority: P0]
**As a** 使用者, **I want** 上櫃股票的分析也記錄分析時價格, **so that** 日後同樣能回測驗證。

```gherkin
Scenario: 記錄櫃買中心收盤價
  Given 櫃買中心公布 6182 在 2026-10-06 的收盤價為 128.00
  When 6182（台股）的分析完成
  Then 分析時價格為 128.00，計價幣別 TWD，價格時間 2026-10-06（台北時間），價格來源「櫃買中心」

Scenario: 上櫃股票當日無成交
  Given 櫃買中心資料中 6182 當日沒有收盤價
  When 6182（台股）的分析完成
  Then 分析事件狀態為「已完成」
  And 分析時價格為「未取得」

Scenario: 上市股票價格來源不變
  Given 證交所公布 2330 的收盤價為 2575.00
  When 2330（台股）的分析完成
  Then 分析時價格為 2575.00，價格來源「證交所」
```

---

## 4. Business Flow & Logic

- **Core Business Rules:**
  - 台股標的辨識順序：證交所上市公司 → 櫃買中心上櫃股票；先找到者為準。
  - 任一邊找到即辨識成功；兩邊都明確沒有 → 找不到此標的；有一邊取不到資料且另一邊沒找到 → 新聞來源暫時無法使用。
  - 台股分析時價格依同樣順序取價：證交所有該股收盤價則用證交所，否則用櫃買中心。
  - 櫃買中心資料（上櫃股票每日收盤行情）保留 1 小時後重新取得；失敗時沿用上一次成功的資料；從未成功過則視為取不到。
  - 櫃買中心收盤價欄位不是數字（例如當日無成交）→ 該股無價格。
- **Edge Cases:**
  - 櫃買中心回傳空資料或格式錯誤 → 視為取不到資料。
  - 櫃買中心資料中沒有代號或沒有公司名稱的資料列不納入（無法用來搜尋新聞，也不提供價格）。
  - 某檔股票的交易日期無法解析 → 該股無價格（與證交所一致）。

---

## 5. UI/UX Design & Interaction

N/A。

---

## 6. Non-Functional Requirements

- **Performance:** 櫃買中心資料約 5 MB，與其他外部來源相同最多等待 10 秒；1 小時快取避免每次查詢都重新下載。
- **Compatibility:** 上市股票的辨識與價格結果與現行完全相同。

---

## 7. Dependencies & Risks

- **External Dependencies:** 櫃買中心開放資料（上櫃股票每日收盤行情）。
- **Known Risks:** 櫃買中心的上櫃公司基本資料端點已失效，公司簡稱改以每日收盤行情的名稱欄位取得；該資料也包含 ETF 等其他上櫃證券，以其代號查詢時會以基金名稱搜尋新聞。

---

## 8. Appendix

- `.sdd/2026-10-07-otc-stock-support/BRIEF.md`
- `.sdd/2026-10-06-symbol-news-search/PRD.md`、`.sdd/2026-10-06-analysis-price-snapshot/PRD.md`
