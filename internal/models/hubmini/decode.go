package hubmini

import (
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type HubMini struct {
	LatestTime uint64 `json:"latest_time"`
}

func DecodeManufacturer(p []byte) protocol.Reading {
	var latest uint64
	for _, value := range p[8:13] {
		latest = latest<<8 | uint64(value)
	}
	return protocol.Reading{Data: HubMini{LatestTime: latest}}
}
