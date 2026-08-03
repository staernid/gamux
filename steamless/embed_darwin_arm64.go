//go:build darwin && arm64

package steamless

import _ "embed"

//go:embed bin/steamless-macos-arm64
var embeddedBinary []byte

const embeddedBinaryName = "steamless"
