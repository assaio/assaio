package cli

import (
	"github.com/assaio/assaio/internal/server"
	"github.com/assaio/assaio/internal/usage"
)

type syncPushRequestV2 struct {
	Protocol     int                   `json:"protocol"`
	MemberDigest string                `json:"memberDigest"`
	Records      []server.SyncRecordV2 `json:"records"`
}

func newSyncPushRequestV2(member string, records []usage.Record) syncPushRequestV2 {
	request := syncPushRequestV2{Protocol: 2, MemberDigest: member, Records: make([]server.SyncRecordV2, len(records))}
	for i := range records {
		request.Records[i] = server.NewSyncRecordV2(&records[i])
	}
	return request
}
