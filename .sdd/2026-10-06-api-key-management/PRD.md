# Product Requirements Document (PRD) — API Key 管理

**Status:** Finalized
**Version:** v1.0
**Owner:** James Hsueh
**Stakeholders:** Engineering

---

## 1. Background & Goal (Why & Goal)

- **Problem Statement:** 服務要對外開放，但分析會消耗 AI 費用，不能讓任何人都直接使用。需要一個讓使用者自助申請、由 administrator 把關啟用、使用者可自行丟棄的 API key 機制。
- **Expected Outcome:**
  - 使用者不需人工介入即可申請 API key，並能自行查詢是否已被啟用。
  - 未經 administrator 啟用、或已撤銷的 API key，100% 無法使用受保護功能。
  - 系統不保存可還原的完整 API key；資料外洩時無法從資料還原出可用的 API key。
- **Out of Scope:**
  - Administrator 啟用 / 停用 API key 的操作介面（直接修改資料完成）。
  - 使用者帳號、登入、API key 清單管理。
  - 申請頻率限制、用量計次、API key 到期時間。
  - 找回遺失的 API key。

---

## 2. User Personas

- **Primary Role(s):**
  - **使用者**：以程式呼叫本服務的外部開發者，持有 API key 即代表擁有者。
  - **Administrator**：專案擁有者，以直接修改資料的方式啟用 / 停用 API key。
- **Usage Context:** 使用者透過自己的程式或命令列工具呼叫服務；無圖形介面。

---

## 3. User Stories & Acceptance Criteria

### US-01 — 申請 API key [priority: P0]
**As a** 使用者, **I want** 提供名稱即可申請 API key, **so that** 我能在 administrator 啟用後開始使用服務。

```gherkin
Scenario: 以有效名稱申請成功
  Given 使用者提供 API key 名稱「我的研究腳本」
  When 使用者申請 API key
  Then 申請成功，使用者取得完整 API key
  And 這把 API key 的狀態為「停用中」

Scenario: 名稱前後的空白會被去除
  Given 使用者提供 API key 名稱「  研究  」
  When 使用者申請 API key
  Then 申請成功，API key 名稱記為「研究」

Scenario: 名稱剛好 100 個字可以申請
  Given 使用者提供長度剛好 100 個字的 API key 名稱
  When 使用者申請 API key
  Then 申請成功

Scenario: 名稱超過 100 個字被拒
  Given 使用者提供長度 101 個字的 API key 名稱
  When 使用者申請 API key
  Then 申請被拒，告知「API key 名稱不可超過 100 個字」

Scenario: 未提供名稱被拒
  Given 使用者沒有提供 API key 名稱
  When 使用者申請 API key
  Then 申請被拒，告知「API key 名稱為必填」

Scenario: 名稱只有空白被拒
  Given 使用者提供的 API key 名稱只有空白
  When 使用者申請 API key
  Then 申請被拒，告知「API key 名稱為必填」

Scenario: 名稱可與其他 API key 重複
  Given 已有一把名稱為「研究」的 API key
  When 使用者以名稱「研究」申請 API key
  Then 申請成功，取得一把與既有 API key 不同的新 API key
```

### US-02 — 完整 API key 只提供一次 [priority: P0]
**As a** 使用者, **I want** 只在申請當下看到完整 API key, **so that** API key 不會在之後被任何人從系統中取回。

```gherkin
Scenario: 申請當下看得到完整 API key
  Given 使用者剛申請成功
  When 使用者查看申請結果
  Then 申請結果包含完整 API key

Scenario: 之後查詢看不到完整 API key
  Given 使用者持有一把已申請的 API key
  When 使用者查詢這把 API key 的狀態
  Then 結果只包含 API key 名稱與狀態
  And 結果不包含完整 API key
```

### US-03 — 查詢 API key 狀態 [priority: P0]
**As a** 使用者, **I want** 查詢自己 API key 的狀態, **so that** 我知道 administrator 是否已經啟用。

```gherkin
Scenario: 尚未啟用的 API key 顯示停用中
  Given 使用者持有一把尚未啟用的 API key
  When 使用者查詢狀態
  Then 得知狀態為「停用中」

Scenario: 已啟用的 API key 顯示已啟用
  Given 使用者持有一把已被 administrator 啟用的 API key
  When 使用者查詢狀態
  Then 得知狀態為「已啟用」

Scenario: 已撤銷的 API key 無法查詢
  Given 使用者持有一把已撤銷的 API key
  When 使用者查詢狀態
  Then 查詢被拒，告知「API key 無效」

Scenario: 不存在的 API key 無法查詢
  Given 使用者提供一把不存在的 API key
  When 使用者查詢狀態
  Then 查詢被拒，告知「API key 無效」

Scenario: 未提供 API key 無法查詢
  Given 使用者沒有提供 API key
  When 使用者查詢狀態
  Then 查詢被拒，告知「需要提供 API key」
```

### US-04 — 撤銷 API key [priority: P0]
**As a** 使用者, **I want** 永久撤銷我的 API key, **so that** API key 外洩或不再需要時，任何人都無法再用它。

```gherkin
Scenario: 撤銷已啟用的 API key
  Given 使用者持有一把已啟用的 API key
  When 使用者撤銷這把 API key
  Then 撤銷成功
  And 之後以這把 API key 使用受保護功能會被拒，告知「API key 無效」

Scenario: 尚未啟用也能撤銷
  Given 使用者持有一把尚未啟用的 API key
  When 使用者撤銷這把 API key
  Then 撤銷成功

Scenario: 已撤銷的 API key 不能再撤銷
  Given 一把已撤銷的 API key
  When 使用者再次撤銷這把 API key
  Then 撤銷被拒，告知「API key 無效」

Scenario: 撤銷後即使 administrator 重新啟用仍無法使用
  Given 一把已撤銷的 API key
  And administrator 之後把它設為啟用
  When 使用者以這把 API key 使用受保護功能
  Then 被拒，告知「API key 無效」

Scenario: 不存在的 API key 不能撤銷
  Given 使用者提供一把不存在的 API key
  When 使用者撤銷這把 API key
  Then 撤銷被拒，告知「API key 無效」
```

### US-05 — 受保護功能的把關 [priority: P0]
**As a** administrator, **I want** 受保護功能只接受已啟用且未撤銷的 API key, **so that** 只有我核准的使用者能消耗服務資源。

```gherkin
Scenario: 已啟用且未撤銷的 API key 放行
  Given 使用者持有一把已啟用、未撤銷的 API key
  When 使用者使用受保護功能
  Then 受保護功能照常執行

Scenario: 停用中的 API key 被拒
  Given 使用者持有一把停用中的 API key（從未啟用，或被 administrator 停用）
  When 使用者使用受保護功能
  Then 被拒，告知「API key 尚未啟用」

Scenario: 已撤銷的 API key 被拒
  Given 使用者持有一把已撤銷的 API key
  When 使用者使用受保護功能
  Then 被拒，告知「API key 無效」

Scenario: 不存在的 API key 被拒
  Given 使用者提供一把不存在的 API key
  When 使用者使用受保護功能
  Then 被拒，告知「API key 無效」

Scenario: 未提供 API key 被拒
  Given 使用者沒有提供 API key
  When 使用者使用受保護功能
  Then 被拒，告知「需要提供 API key」
```

---

## 4. Business Flow & Logic

- **Flow:**
  1. 使用者提供名稱申請 → 取得完整 API key（僅此一次），狀態「停用中」。
  2. Administrator 直接修改資料啟用 → 狀態「已啟用」。
  3. 使用者持 API key 使用受保護功能；可隨時查詢狀態。
  4. 使用者撤銷 → 永久失效。
- **Core Business Rules:**
  - API key 狀態：停用中 ⇄ 已啟用（由 administrator 切換）；任一狀態皆可 → 已撤銷（由持有者觸發，不可逆）。
  - 已撤銷優先於啟用狀態：已撤銷的 API key 無論是否啟用，一律視為無效。
  - API key 名稱：必填，去除前後空白後長度 1–100 字，可重複。
  - 「無效」涵蓋「不存在」與「已撤銷」兩種情況，對外不區分，避免洩漏 API key 是否曾經存在。
- **Edge Cases:**
  - 申請過程中資料無法保存 → 申請失敗，使用者不會取得任何 API key。
  - 驗證 API key 時資料無法讀取 → 拒絕本次請求並告知服務暫時無法使用，不可放行。

---

## 5. UI/UX Design & Interaction

N/A — 無圖形介面，僅供程式呼叫。

---

## 6. Non-Functional Requirements

- **Performance:** 驗證 API key 會發生在每次受保護功能的請求上，單次驗證不應明顯增加回應時間。
- **Security:**
  - 系統不保存可還原的完整 API key。
  - 完整 API key 需具備足夠的隨機性，無法被猜測或列舉。
  - 完整 API key 帶有可辨識本服務的固定開頭，方便使用者辨認與外洩掃描。
- **Compatibility:** N/A
- **Analytics / Tracking:** N/A

---

## 7. Dependencies & Risks

- **External Dependencies:** 無。
- **Known Risks:**
  - 申請功能公開且無頻率限制，可能被大量灌入申請；目前僅靠「預設停用」把關。

---

## 8. Appendix

- `.sdd/2026-10-06-api-key-management/BRIEF.md`
- `.sdd/PROJECT.md`、`.sdd/UL-MAP.md`
