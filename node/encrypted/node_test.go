package encrypted

import (
	"testing"

	"github.com/FluffyKebab/pearly/node/basic"
	"github.com/FluffyKebab/pearly/testutil"
	"github.com/FluffyKebab/pearly/transport/tcp"
	"github.com/stretchr/testify/require"
)

func createEnctyptedPingNode(t *testing.T) {
	t.Helper()

	port, err := testutil.GetAvilablePort()
	require.NoError(t, err)

	node, err := New(basic.New(tcp.New(port)))
	require.NoError(t, err)

}

func TestEncryptedPing(t *testing.T) {

}
