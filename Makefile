all:
	go test -v -cover=true ./snowflake_test.go ./snowflake_go.go

# 测试单个用例
succ:
	 go test -v -cover=true ./snowflake_test.go ./snowflake_go.go -run TestSnowflakeSucc

# 测试单个用例
fail:
	go test -v -cover=true ./snowflake_test.go ./snowflake_go.go -run TestSnowflakeFail

# 测试整个包
module:
	cd ..; go test -v -cover=true ./snowflake
