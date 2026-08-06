package models

type RadiusUser struct {
	ID              int    `json:"id"`
	Username        string `json:"username"`
	Password        string `json:"password,omitempty"`
	SimultaneousUse int    `json:"simultaneous_use"`
	Group           string `json:"group"`
}

type ActiveSession struct {
	UniqueID   string `json:"unique_id"`
	Username   string `json:"username"`
	RemoteIP   string `json:"remote_ip"`
	RemoteID   string `json:"remote_id"`
	VirtualIP  string `json:"virtual_ip"`
	State      string `json:"state"`
	IKEVersion string `json:"ike_version"`
	StartTime  string `json:"start_time"`
	BytesIn    uint64 `json:"bytes_in"`
	BytesOut   uint64 `json:"bytes_out"`
}

type AppSetting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
