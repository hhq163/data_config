package impl

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"dataconfig/base"
	_ "dataconfig/output"

	"github.com/golang/protobuf/proto"
	"github.com/hhq163/logger"
	"github.com/tealeg/xlsx"
)

//
/**
 * ProtoToBytes 将excel文件中的数据序列成pb格式
 * inputDir excel文件目录
 * outputDir 目标目录
 */
func ProtoToBytes(inputDir, outputDir string) {
	tLog := base.Log.With(logger.FuncName, "ProtoToBytes")
	tLog.Debug("in")

	absPath, _ := filepath.Abs(base.GetExecpath() + "/" + outputDir)
	err := filepath.Walk(absPath, func(path string, fi os.FileInfo, err error) error {
		if nil == fi {
			return err
		}

		if fi.IsDir() {
			return nil
		}
		name := fi.Name()

		match, _ := regexp.MatchString("(.*).bytes", name)
		if match {
			p := filepath.Dir(path)
			os.Remove(p + "/" + name)
		}

		return nil
	})

	files, err := ioutil.ReadDir(base.GetExecpath() + "/" + inputDir)
	if err != nil {
		tLog.Error("input file error: ", inputDir, err.Error())
		return
	}

	for _, file := range files {
		fileAllName := file.Name()
		xlFile, err := xlsx.OpenFile(inputDir + "/" + fileAllName)
		if err != nil {
			fmt.Printf("open file fileName=%s", fileAllName)
			tLog.Error("config is wrong!!!", fileAllName, ",err=", err.Error())
			continue
		}

		if len(xlFile.Sheets) == 0 {
			continue
		}

		for key, sheet := range xlFile.Sheets {
			fileName := getFileName(sheet.Name)

			if fileName == "" {
				fmt.Printf("sheet.Name is empty fileAllName=%s, key=%d", fileAllName, key)
				tLog.Error("sheet.Name is empty fileAllName=", fileAllName, ",key=", key)
				continue
			}

			match, _ := regexp.MatchString("[a-zA-Z.]", fileName)
			if !match {
				tLog.Info("sheet.Name is not english fileName=", fileName, ",key=", key)
				continue
			}

			if len(sheet.Rows) < 3 {
				fmt.Printf("file is empty fileName=%s, key=%d", fileName, key)
				tLog.Error("file is empty fileName=", fileName, ",key=", key)
				continue
			}

			configStr := fmt.Sprintf("output.%sConfigData", fileName) //output.PaymentSettingsConfigData
			dataStr := fmt.Sprintf("output.%s", fileName)             //output.PaymentSettings
			dataType := proto.MessageType(dataStr)
			configType := proto.MessageType(configStr)
			fmt.Println("111111111: ", dataStr, configStr, dataType, configType)

			configObj := reflect.New(configType.Elem()).Elem()
			rowNum := sheet.MaxRow
			dataList := reflect.MakeSlice(reflect.SliceOf(dataType), 0, rowNum)

			row1 := sheet.Rows[1] //第二行 类型
			row2 := sheet.Rows[2] // 第三行 paramName
			nameParamMap := getNameMap(row1, row2)

			for i := 3; i < rowNum; i++ { //从第四行开始是数据
				rowData := sheet.Rows[i]
				dataObj := reflect.New(dataType.Elem()).Elem()
				row1CellLen := len(row1.Cells)
				if row1CellLen > dataObj.NumField()-3 { //3个protoc额外加的字段
					row1CellLen = dataObj.NumField() - 3
				}
				tLog.Debug("dataObj filed len=", dataObj.NumField(), ",row1CellLen=", row1CellLen)
				findex := 0
				for j := 0; j < row1CellLen-1; j++ {
					data := rowData.Cells[j].Value
					typeStr := strings.ToLower(row1.Cells[j].String())
					if typeStr == "" {
						tLog.Info("typeStr is empty sheetName=", fileName, ",key=", key, ",i=", i, ", j=", j)
						continue
					}

					if filedIndex, ok := nameParamMap[row2.Cells[j].String()]; ok {
						findex = filedIndex - 1
					}
					// tLog.Debug("row2.Cells[j].String()=", row2.Cells[j].String(), ",findex=", findex)

					switch typeStr {
					case "string":
						dataObj.Field(findex).SetString(data)
					case "integer":
						x := ToInt64(data)
						dataObj.Field(findex).SetInt(x)
					case "array":
						strList := strings.Split(data, ",")
						var sli []int32
						for _, v := range strList {
							sli = append(sli, ToInt32(v))
						}
						dataObj.Field(findex).Set(reflect.ValueOf(sli))
					case "float":
						dataObj.Field(findex).SetFloat(ToFloat(data))
					default:
						fmt.Println("file error : ", file.Name(), row1.Cells[findex].String(), ",typeStr=", typeStr)
						panic("file error: " + file.Name())
					}
				}
				// tLog.Debug("ttttttttttttt2 : ", dataObj, ", compare=", dataObj.Addr().Kind() == reflect.Ptr)
				dataList = reflect.Append(dataList, dataObj.Addr())
			}

			// tLog.Debug("datalist: ", dataList, configObj.Field(0).Kind() == reflect.Slice)
			configObj.Field(0).Set(dataList)
			pb := configObj.Addr().Interface().(proto.Message)
			b, err := proto.Marshal(pb)
			if err != nil {
				panic("proto data Marshal fail!")
			}
			outPath := outputDir + "/" + fileName + ".bytes"
			tLog.Debug("outpath: ", outPath)
			ioutil.WriteFile(outPath, b, os.ModePerm)

		}
	}

}

//将proto中序号、字段名封装成map[序号]字段名
func getNameMap(typeRow, paramNameRow *xlsx.Row) map[string]int {
	tLog := base.Log.With(logger.FuncName, "getNameMap")
	tLog.Debug("in")

	row1len := len(typeRow.Cells)
	ret := make(map[string]int, 0)
	num := 0

	for k, v := range paramNameRow.Cells {
		if k > row1len-1 {
			continue
		}

		paramStr := v.String()
		if paramStr == "" {
			continue
		}

		typeStr := strings.ToLower(typeRow.Cells[k].String())
		paramStrtmp := strings.ToLower(paramStr)
		if typeStr == "" && (paramStrtmp == "key" || paramStrtmp == "key1" || paramStrtmp == "key2") {
			typeStr = "integer"
		}
		if typeStr == "" {
			tLog.Info("typeStr is empty ,k=", k)
			continue
		}

		num++
		ret[paramStr] = num
	}
	tLog.Debugf("getNameMap=%v", ret)
	tLog.Debug("end")
	return ret
}
