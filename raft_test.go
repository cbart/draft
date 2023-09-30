package draft_test

import (
	"context"
	"testing"
	"time"

	"github.com/cbart/draft"
)

func TestF(t *testing.T) {
	n1, n2, n3 := draft.Follower{}, draft.Follower{}, draft.Follower{}
	cluster := draft.Cluster{n1.LocalClient(), n2.LocalClient(), n3.LocalClient()}
	ctx := context.Background()
	if err := cluster.Append(ctx, "example"); err != nil {
		t.Fatalf("failed to append: %s", err)
	}
	consensus := n1.AwaitLastLog("example")
	defer consensus.Cancel()
	select {
	case <-time.After(1 * time.Second):
		t.Error("timed out waiting for consensus")
	case <-consensus:
	}
}
