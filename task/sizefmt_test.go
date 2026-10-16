package task

import (
	"strconv"
	"testing"
)

func TestSizeFmt(t *testing.T) {
	tab := []struct {
		size int64
		want string
	}{
		{0, "0"},
		{100, "100"},
		{1000, "1000"},
		{1024, "1Ki"},
		{1025, "1.001Ki"},
		{2048, "2Ki"},
		{65536, "64Ki"},
		{1024*1024 + 0, "1Mi"},
		{1024*1024 + 1000, "1.001Mi"},
		{1024*1024*1024 + 0, "1Gi"},
		{1024*1024*1024*1024 + 0, "1024Gi"},
	}

	for i, tc := range tab {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			s := sizeFmt(tc.size)
			if s != tc.want {
				t.Errorf("mismatch\nhave %q\nwant %q", s, tc.want)
			}
		})
	}
}
