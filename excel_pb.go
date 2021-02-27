package data_config

import (
	"github.com/hhq163/data_config/base"
	"github.com/hhq163/data_config/impl"
)

func ExcelToPb(input, output string, protoVer int32) {
	base.LogInit(true, "github.com/hhq163/data_config")

	impl.ExcelToProto(input, output, protoVer)
	impl.ProtoToBytes(input, output)
}
