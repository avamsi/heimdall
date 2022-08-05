package bifrost

type NotifyRequest struct {
	Cmd         string `json:"cmd"`
	StartTime   int    `json:"start_time"`
	Code        int    `json:"code"`
	ForceNotify bool   `json:"force_notify"`
}
