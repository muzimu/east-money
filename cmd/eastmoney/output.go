package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
)

const (
	formatJSON  = "json"
	formatHuman = "human"
)

// printOutput 输出数据到 stdout，根据 --format flag 选择 JSON 或可读格式。
func printOutput(v any) {
	FprintOutput(flagFormat, os.Stdout, v)
}

// FprintOutput 按指定格式将数据写入 w。
func FprintOutput(format string, w io.Writer, data any) {
	switch format {
	case formatHuman:
		fprintHuman(w, data)
	case formatJSON:
		fprintJSON(w, data)
	default:
		fprintJSON(w, data)
	}
}

func fprintJSON(w io.Writer, data any) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(w, "%+v\n", data)
		return
	}
	fmt.Fprintln(w, string(b))
}

func fprintHuman(w io.Writer, data any) {
	v := reflect.ValueOf(data)

	// 处理指针
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			fmt.Fprintln(w, "(nil)")
			return
		}
		v = v.Elem()
	}

	// 尝试展开 APIResponse wrapper：跳过 success/error 包装，直接展示 Data
	if v.Kind() == reflect.Struct {
		if inner, ok := tryUnwrapResponse(v); ok {
			fprintHuman(w, inner)
			return
		}
	}

	switch v.Kind() {
	case reflect.Slice:
		printTable(w, v)
	case reflect.Struct:
		printStruct(w, v)
	default:
		fmt.Fprintf(w, "%v\n", data)
	}
}

// tryUnwrapResponse 检测 APIResponse 包装结构并提取 Data 字段。
// 成功且 Data 非 nil → 返回 Data，用于递归展开。
// 失败 → 返回 Error 信息作为可打印字符串。
// 不是包装类型 → 返回 false。
func tryUnwrapResponse(v reflect.Value) (any, bool) {
	successField := v.FieldByName("Success")
	dataField := v.FieldByName("Data")
	if !successField.IsValid() || !dataField.IsValid() {
		return nil, false
	}

	if !successField.Bool() {
		// 失败：展示错误信息
		errField := v.FieldByName("Error")
		if errField.IsValid() && !errField.IsNil() {
			return errField.Interface(), true
		}
		return nil, false
	}

	// 成功：提取 Data
	if dataField.IsNil() {
		return "(empty)", true
	}
	return dataField.Elem().Interface(), true
}

// printStruct 将单个结构体打印为 key-value 格式。
// 结构体切片字段自动展开为嵌套表格。
func printStruct(w io.Writer, v reflect.Value) {
	t := v.Type()
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	type nestedSlice struct {
		name string
		v    reflect.Value
	}
	var nested []nestedSlice

	for i := 0; i < t.NumField(); i++ {
		field := v.Field(i)
		if !field.IsValid() {
			continue
		}
		name := jsonTagName(t.Field(i))
		if name == "-" {
			continue
		}

		// 结构体切片 → 收集后展开为嵌套表格
		if isStructSlice(field) {
			nested = append(nested, nestedSlice{name, field})
			continue
		}

		value := formatFieldValue(field)
		fmt.Fprintf(tw, "%s:\t%s\n", name, value)
	}
	tw.Flush()

	// 嵌套切片在标量字段之后展开
	for _, ns := range nested {
		fmt.Fprintf(w, "\n%s:\n", ns.name)
		printTable(w, ns.v)
	}
}

// printTable 将结构体切片打印为表格。
func printTable(w io.Writer, v reflect.Value) {
	if v.Len() == 0 {
		fmt.Fprintln(w, "(empty)")
		return
	}

	first := v.Index(0)
	if first.Kind() == reflect.Ptr {
		if first.IsNil() {
			fmt.Fprintln(w, "(empty)")
			return
		}
		first = first.Elem()
	}
	if first.Kind() != reflect.Struct {
		fprintJSON(w, v.Interface())
		return
	}

	t := first.Type()
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// 表头
	headers := make([]string, 0, t.NumField())
	cols := make([]int, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		name := jsonTagName(t.Field(i))
		if name == "-" {
			continue
		}
		headers = append(headers, name)
		cols = append(cols, i)
	}
	if len(headers) == 0 {
		return
	}
	fmt.Fprintln(tw, strings.Join(headers, "\t"))

	// 数据行
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		if item.Kind() == reflect.Ptr {
			if item.IsNil() {
				continue
			}
			item = item.Elem()
		}
		row := make([]string, len(cols))
		for j, col := range cols {
			row[j] = formatFieldValue(item.Field(col))
		}
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	tw.Flush()
}

// jsonTagName 从 struct field 的 json tag 中提取字段名。
func jsonTagName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}
	name, _, _ := strings.Cut(tag, ",")
	return name
}

// formatFieldValue 格式化字段值。
func formatFieldValue(field reflect.Value) string {
	if !field.IsValid() {
		return "-"
	}
	switch {
	case field.Kind() == reflect.Ptr && !field.IsNil():
		return fmt.Sprintf("%v", field.Elem().Interface())
	case field.Kind() == reflect.Ptr && field.IsNil():
		return "-"
	case field.Kind() == reflect.Slice:
		return fmt.Sprintf("[%d items]", field.Len())
	default:
		return fmt.Sprintf("%v", field.Interface())
	}
}

// isStructSlice 判断是否为结构体切片（可展开为嵌套表格）。
func isStructSlice(v reflect.Value) bool {
	if v.Kind() != reflect.Slice || v.Len() == 0 {
		return false
	}
	first := v.Index(0)
	if first.Kind() == reflect.Ptr {
		if first.IsNil() {
			return false
		}
		first = first.Elem()
	}
	return first.Kind() == reflect.Struct
}
