package vo

type AnalystRequestVo struct {
	Symbol        string
	Category      string
	SearchKeyword string
	// a zero window means the analyst is not limited to a publishing period
	NewsPublishedWindow NewsPublishedWindowVo
}
