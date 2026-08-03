//go:build linux && arm64

package steamless

import _ "embed"

//go:embed bin/steamless-linux-arm64
var embeddedBinary []byte

const embeddedBinaryName = "steamless"
