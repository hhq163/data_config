package dataconfig

import (
	"dataconfig/base"
	"dataconfig/impl"
)

func ExcelToPb(input, output string, protoVer int32) {
	base.LogInit(true, "dataconfig")

	impl.ExcelToProto(input, output, protoVer)
	impl.ProtoToBytes(input, output)
}
