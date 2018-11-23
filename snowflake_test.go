package snowflake

import (
	"testing"
)

func TestSnowflakeFail(t *testing.T) {
	_, err := NewIdWorker(512)
	if err != nil {
		t.Error("faild")
	} else {
		t.Log("pass")
	}
}

func TestSnowflakeSucc(t *testing.T) {
	idworker, err := NewIdWorker(511)
	if err != nil {
		t.Error("faild")
	} else {
		t.Log("pass")
		if _, err := idworker.NextId(); err != nil {
			t.Error("faild")
		} else {
			t.Log("pass")
		}
	}
}

func BenchmarkSnowflake(b *testing.B) {
	idworker, err := NewIdWorker(511)
	if err == nil {
		for i := 0; i != 100000; i++ {
			idworker.NextId()
		}
	}
}
