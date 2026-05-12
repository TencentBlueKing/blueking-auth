package gpu_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGpu(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Realm Gpu Suite")
}
