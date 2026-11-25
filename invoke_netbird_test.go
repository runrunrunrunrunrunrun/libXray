package libXray

import (
	"testing"

	"github.com/xtls/libxray/nodep"
	"github.com/xtls/libxray/xray"
)

func TestInvokeNetBirdOAuthValidation(t *testing.T) {
	// Invalid management URL scheme should fail cleanly.
	res := invokeForTest(t, LibXrayMethodNetBirdStartOAuth, &NetBirdStartOAuthRequest{
		ManagementUrl: "ftp://invalid.example",
	})
	if res.Success || res.Err == "" {
		t.Fatalf("expected failure for invalid scheme, got %+v", res)
	}

	// Missing sessionId on complete should fail cleanly.
	resComplete := invokeForTest(t, LibXrayMethodNetBirdCompleteOAuth, &NetBirdCompleteOAuthRequest{})
	if resComplete.Success || resComplete.Err == "" {
		t.Fatalf("expected failure for missing sessionId, got %+v", resComplete)
	}

	// Cancel on unknown sessionId should succeed idempotently.
	resCancel := invokeForTest(t, LibXrayMethodNetBirdCancelOAuth, &NetBirdCancelOAuthRequest{
		SessionId: "non-existent",
	})
	if !resCancel.Success {
		t.Fatalf("expected cancel to succeed, got %+v", resCancel)
	}
}

func TestPingBatchRejectsNetBirdOutbound(t *testing.T) {
	results, err := xray.PingBatch([]xray.PingBatchItem{
		{
			XrayJSON: `{
				"outbounds": [
					{
						"tag": "nb",
						"protocol": "netbird",
						"settings": {
							"setupKey": "test-key"
						}
					}
				]
			}`,
			OutboundTag: "nb",
		},
	}, 5, "https://example.com")
	if err != nil {
		t.Fatalf("PingBatch returned unexpected top-level error: %v", err)
	}
	if len(results) != 1 || results[0].Success || results[0].Delay != nodep.PingDelayError {
		t.Fatalf("expected failed PingBatchResult for netbird, got %+v", results)
	}
}
