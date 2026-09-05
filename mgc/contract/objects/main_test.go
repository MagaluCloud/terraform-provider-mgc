//go:build contract

package objects_test

import (
	"os"
	"testing"

	ct "github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/contracttest"
)

func TestMain(m *testing.M) { os.Exit(ct.Run(m)) }
