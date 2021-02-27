package impl

import (
	"bytes"
	"dataconfig/base"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/hhq163/logger"
	"github.com/tealeg/xlsx"
)

/**
 * ExcelToProto 根据excel文件生成proto和pb文件
 * inputDir excel文件目录
 * outputDir 目标目录
 * protoVer为proto版本，目前支持2,3
 */
func ExcelToProto(inputDir, outputDir string, protoVer int32) {
	tLog := base.Log.With(logger.FuncName, "ExcelToProto")
	tLog.Debug("in")

	//先清空output中原有proto文件
	absPath, _ := filepath.Abs(base.GetExecpath() + "/" + outputDir)
	err := filepath.Walk(absPath, func(path string, fi os.FileInfo, err error) error {
		if nil == fi {
			return err
		}

		if fi.IsDir() {
			return nil
		}
		name := fi.Name()

		match, _ := regexp.MatchString("(.*).proto", name)
		if match {
			p := filepath.Dir(path)
			os.Remove(p + "/" + name)
		}

		match, _ = regexp.MatchString("(.*).pb.go", name)
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

			if len(sheet.Rows) < 2 {
				fmt.Printf("file is empty fileName=%s, key=%d", fileName, key)
				tLog.Error("file is empty fileName=", fileName, ",key=", key)
				continue
			}

			row0 := sheet.Rows[0] // 第一行,描述
			row1 := sheet.Rows[1] //第二行 类型
			row2 := sheet.Rows[2] // 第三行 paramName
			row1len := len(row1.Cells)

			dataStr, primaryKey := "", ""
			num := 0
			for k, v := range row2.Cells {
				if k > row1len-1 {
					continue
				}

				paramStr := v.String()
				if paramStr == "key" {
					primaryKey = paramStr
				}
				if paramStr == "" {
					continue
				}

				typeStr := strings.ToLower(row1.Cells[k].String())
				paramStrtmp := strings.ToLower(paramStr)
				if typeStr == "" && (paramStrtmp == "key" || paramStrtmp == "key1" || paramStrtmp == "key2") {
					typeStr = "integer"
				}

				if typeStr == "" {
					tLog.Info("typeStr is empty sheetName=", fileName, ",paramStr=", paramStr)
					continue
				}

				descStr := row0.Cells[k].String()

				dataStrTmp, err := genDataStr(typeStr, protoVer)
				if err != nil {
					fmt.Println("err=", err.Error(), "file= ", file.Name(), row2.Cells[k].String(), ",typeStr=", typeStr, ",paramStr=", paramStr)
					panic("file error: " + file.Name())
				}
				num++
				dataStr += dataStrTmp
				dataStr += paramStr + " = " + strconv.Itoa(num) + ";		//" + descStr + "\n"

			}

			fileData := genProtoContent(primaryKey, fileName, dataStr, 3)
			outPath := base.GetExecpath() + "/" + outputDir + "/" + fileName + ".proto"
			tLog.Debug("outPath=", outPath)

			_, err := os.Stat(outPath)
			if err != nil { //文件不存在
				err = ioutil.WriteFile(outPath, []byte(fileData), os.ModePerm)
				if err != nil {
					fmt.Println("save file error! outfile:", outPath)
					tLog.Error("save file error! outfile=", outPath, ", err=", err.Error())
				}

			} else {
				err = appendToFile(outPath, fileData)
				if err != nil {
					fmt.Println("appendToFile file error! outfile=", outPath)
					tLog.Error("appendToFile file error! outfile=", outPath, ", err=", err.Error())
				}
			}

		}

	}

	outPath := base.GetExecpath() + "/" + outputDir
	// outPath := "./" + outputDir

	// cmd := exec.Command("protoc", fmt.Sprintf(" --proto_path=%s", outPath), fmt.Sprintf(" --gofast_out=%s", outPath), fmt.Sprintf(" %s/*.proto", outPath))
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("protoc --proto_path=%s --gofast_out=%s %s/*.proto", outPath, outPath, outPath))

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		tLog.Error("proto 生成工具出错, err=", err.Error(), ",stderr=", stderr.String())
		fmt.Println("proto 生成工具出错 ", err.Error())
		return
	}

	tLog.Debug("end")
}

func genDataStr(typeStr string, protoVer int32) (string, error) {
	dataStr := ""
	if protoVer == 2 {
		switch typeStr {
		case "string":
			dataStr += "		optional string "
		case "integer":
			dataStr += "		optional int32 "
		case "array":
			dataStr += "		repeated int32 "
		case "float":
			dataStr += "		optional float "
		default:
			return "", errors.New("typerStr not identify!")
		}
	} else {
		switch typeStr {
		case "string":
			dataStr += "		string "
		case "integer":
			dataStr += "		int32 "
		case "array":
			dataStr += "		repeated int32 "
		case "float":
			dataStr += "		float "
		default:
			return "", errors.New("typerStr not identify!")
		}
	}
	return dataStr, nil
}

/**
* 生成proto内容
* ver proto版本
 */
func genProtoContent(primaryKey, fileName, dataStr string, ver int) string {
	var content string
	if ver == 2 {
		content = fmt.Sprintf(
			`syntax = "proto2";
package output;
		
// key:["%s"]
message %s {
%s
}
		
message %sConfigData{
	repeated %s config = 1;
}`, primaryKey, fileName, dataStr, fileName, fileName)
	} else {
		content = fmt.Sprintf(
			`syntax = "proto3";
package output;
		
// key:["%s"]
message %s {
%s
}
		
message %sConfigData{
	repeated %s config = 1;
}`, primaryKey, fileName, dataStr, fileName, fileName)

	}

	return content
}
