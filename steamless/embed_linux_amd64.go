//go:build linux && amd64

package steamless

import _ "embed"

//go:embed bin/steamless-linux-x64
var embeddedBinary []byte

const embeddedBinaryName = "steamless"
