package blueking

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBlueking(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Realm Blueking Suite")
}
