# API Key 管理 — Requirements Brief

## Goal
讓外部使用者能自助申請 API key，但只有經 administrator 啟用的 API key 才能使用分析等受保護功能。使用者可隨時撤銷自己的 API key，撤銷後該 API key 永久失效。這是對外提供服務的入場券，也是後續所有受保護功能共用的把關機制。

## Requirements
- 任何人都能申請 API key，申請時必須提供 API key 名稱。
- 新申請的 API key 一律為「停用」，等 administrator 啟用後才算可用。
- 完整的 API key 只在申請成功當下提供一次；之後系統任何地方都不會再顯示完整的 API key（系統本身也不保存可還原的完整 API key）。
- 持有 API key 的人可以查詢該 API key 目前的狀態（停用中 / 已啟用），用來得知 administrator 是否已經啟用。
- 持有 API key 的人可以撤銷它。撤銷是永久的：撤銷後即使 administrator 再把它設為啟用，也不能再使用。
- 受保護功能（如分析）只接受「已啟用且未撤銷」的 API key；其餘情況一律拒絕並說明原因。
- Administrator 啟用 / 停用 API key 是直接修改資料完成，不在本功能提供操作入口。

## Examples (Specification by Example)
Each example lists **only** the data that affects the behavior — nothing more.

### Rule: 申請 API key 必須提供有效名稱
| # | Given (only relevant data) | When | Then |
|---|---|---|---|
| 1 (happy)     | 名稱「我的研究腳本」 | 申請 API key | 申請成功，取得完整 API key，狀態為「停用中」 |
| 2 (boundary)  | 名稱前後帶空白「  研究  」 | 申請 API key | 申請成功，名稱記為「研究」 |
| 3 (boundary)  | 名稱剛好 100 個字 | 申請 API key | 申請成功 |
| 4 (exception) | 名稱 101 個字 | 申請 API key | 申請被拒，告知名稱過長 |
| 5 (exception) | 未提供名稱，或名稱只有空白 | 申請 API key | 申請被拒，告知名稱為必填 |
| 6 (boundary)  | 已有另一把 API key 也叫「研究」 | 以名稱「研究」申請 | 申請成功，兩把 API key 互不影響 |

### Rule: 完整 API key 只提供一次
| # | Given (only relevant data) | When | Then |
|---|---|---|---|
| 1 (happy)     | 剛申請成功 | 查看申請結果 | 看得到完整 API key |
| 2 (exception) | 已申請過的 API key | 查詢 API key 狀態 | 只看得到名稱與狀態，看不到完整 API key |

### Rule: 查詢 API key 狀態
| # | Given (only relevant data) | When | Then |
|---|---|---|---|
| 1 (happy)     | 持有尚未啟用的 API key | 查詢狀態 | 得知狀態為「停用中」 |
| 2 (happy)     | 持有已被 administrator 啟用的 API key | 查詢狀態 | 得知狀態為「已啟用」 |
| 3 (exception) | 持有已撤銷的 API key | 查詢狀態 | 被拒，告知 API key 無效 |
| 4 (exception) | 提供不存在的 API key | 查詢狀態 | 被拒，告知 API key 無效 |
| 5 (exception) | 未提供 API key | 查詢狀態 | 被拒，告知需要提供 API key |

### Rule: 撤銷 API key 是永久的
| # | Given (only relevant data) | When | Then |
|---|---|---|---|
| 1 (happy)     | 持有已啟用的 API key | 撤銷 | 撤銷成功；之後此 API key 無法再使用任何功能 |
| 2 (boundary)  | 持有尚未啟用的 API key | 撤銷 | 撤銷成功（不必等啟用也能丟棄） |
| 3 (exception) | 已撤銷的 API key | 再次撤銷 | 被拒，告知 API key 無效 |
| 4 (exception) | 已撤銷的 API key，administrator 之後又把它設為啟用 | 使用受保護功能 | 被拒，告知 API key 無效 |
| 5 (exception) | 提供不存在的 API key | 撤銷 | 被拒，告知 API key 無效 |

### Rule: 受保護功能只接受已啟用且未撤銷的 API key
| # | Given (only relevant data) | When | Then |
|---|---|---|---|
| 1 (happy)     | 已啟用、未撤銷 | 使用受保護功能 | 放行 |
| 2 (exception) | 停用中（從未啟用，或被 administrator 停用） | 使用受保護功能 | 被拒，告知 API key 尚未啟用 |
| 3 (exception) | 已撤銷 | 使用受保護功能 | 被拒，告知 API key 無效 |
| 4 (exception) | 不存在的 API key | 使用受保護功能 | 被拒，告知 API key 無效 |
| 5 (exception) | 未提供 API key | 使用受保護功能 | 被拒，告知需要提供 API key |

## Out of Scope
- Administrator 啟用 / 停用 API key 的操作介面（直接修改資料完成）。
- 使用者帳號、登入、一個使用者管理多把 API key 的清單。
- 申請 API key 的頻率限制、用量計次、到期時間。
- 找回遺失的 API key（遺失只能重新申請）。

## Open Decisions
Items the PRD author should resolve:
- 使用者如何在請求中出示 API key。
- 完整 API key 的格式（長度、可辨識的前綴）。

## Context / Background
- 使用者決策：建立 API key 公開、預設停用；administrator 直接修改資料啟用；另有撤銷功能。
- 本專案無帳號系統，「持有 API key」即代表擁有者，因此查詢狀態與撤銷都以出示 API key 為憑。
- 撤銷設計為不可逆，以免 administrator 誤把已丟棄的 API key 重新打開。
- 申請端點公開且無頻率限制是已知風險，目前僅靠「預設停用」把關。
