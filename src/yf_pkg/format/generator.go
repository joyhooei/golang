package format

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
)

type JSON string

func genJSON(data interface{}, buf *bytes.Buffer) (err error) {
	typ := reflect.TypeOf(data)
	value := reflect.ValueOf(data)
	switch {
	case strings.Index(typ.String(), "map[string]") == 0:
		buf.WriteString("{")
		keys := value.MapKeys()
		for i, k := range keys {
			if i != 0 {
				buf.WriteString(",")
			}
			buf.WriteString(fmt.Sprintf("\"%v\":", k))
			genJSON(value.MapIndex(k).Interface(), buf)
		}
		buf.WriteString("}")
	case strings.Index(typ.String(), "[]") == 0:
		buf.WriteString("[")
		for i := 0; i < value.Len(); i++ {
			if i != 0 {
				buf.WriteString(",")
			}
			genJSON(value.Index(i).Interface(), buf)
		}
		buf.WriteString("]")
	case typ.String() == "string":
		buf.WriteString(fmt.Sprintf("\"%v\"", strings.Replace(value.String(), "\"", "\\\"", -1)))
	case typ.String() == "JSON" || typ.String() == "format.JSON":
		buf.WriteString(fmt.Sprintf("%v", data))
	case strings.Index(typ.String(), "int") >= 0:
		buf.WriteString(fmt.Sprintf("%v", data))
	case strings.Index(typ.String(), "float") >= 0:
		buf.WriteString(fmt.Sprintf("%v", data))
	default:
		buf.WriteString(fmt.Sprintf("\"%v\"", data))
	}
	return
}

func GenerateJSON(data interface{}) (output JSON) {
	var buf bytes.Buffer
	err := genJSON(data, &buf)
	if err != nil {
		buf.Truncate(0)
		buf.WriteString("{\"status\":\"fail\",\"error\":\"" + strings.Replace(err.Error(), "\"", "\\\"", -1) + "\"}")
	}
	return JSON(buf.String())
}
