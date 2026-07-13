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
	if v.Kind() == reflect.Pointer {
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
	case reflect.Map:
		printMap(w, v)
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

// printMap 将 map 打印为 key-value 格式。
func printMap(w io.Writer, v reflect.Value) {
	if v.Len() == 0 {
		fmt.Fprintln(w, "(empty)")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, key := range v.MapKeys() {
		name := fmt.Sprintf("%v", key.Interface())
		value := formatFieldValue(v.MapIndex(key))
		fmt.Fprintf(tw, "%s:\t%s\n", name, value)
	}
	tw.Flush()
}

// printStruct 将单个结构体打印为 key-value 格式。
// 结构体切片字段自动展开为嵌套表格，结构体指针字段递归展开。
func printStruct(w io.Writer, v reflect.Value) {
	t := v.Type()
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	type nestedBlock struct {
		name    string
		v       reflect.Value
		isSlice bool
	}
	var nested []nestedBlock

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
			nested = append(nested, nestedBlock{name, field, true})
			continue
		}

		// 结构体指针 → 收集后递归展开
		if isStructPtr(field) {
			nested = append(nested, nestedBlock{name, field.Elem(), false})
			continue
		}

		value := formatFieldValue(field)
		fmt.Fprintf(tw, "%s:\t%s\n", name, value)
	}
	tw.Flush()

	// 嵌套块在标量字段之后展开
	for _, nb := range nested {
		fmt.Fprintf(w, "\n%s:\n", nb.name)
		if nb.isSlice {
			printTable(w, nb.v)
		} else {
			printStruct(w, nb.v)
		}
	}
}

// isStructPtr 判断是否为非 nil 的结构体指针。
func isStructPtr(v reflect.Value) bool {
	return v.Kind() == reflect.Pointer && !v.IsNil() && v.Elem().Kind() == reflect.Struct
}

// printTable 将结构体切片打印为对齐表格（支持 CJK 字符宽度）。
func printTable(w io.Writer, v reflect.Value) {
	if v.Len() == 0 {
		fmt.Fprintln(w, "(empty)")
		return
	}

	first := v.Index(0)
	if first.Kind() == reflect.Pointer {
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

	// 收集列信息
	cols := make([]int, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		name := jsonTagName(t.Field(i))
		if name == "-" {
			continue
		}
		cols = append(cols, i)
	}
	if len(cols) == 0 {
		return
	}

	// 构建所有行（表头 + 数据）
	rows := make([][]string, 0, v.Len()+1)
	headers := make([]string, len(cols))
	for j, col := range cols {
		headers[j] = jsonTagName(t.Field(col))
	}
	rows = append(rows, headers)

	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		if item.Kind() == reflect.Pointer {
			if item.IsNil() {
				continue
			}
			item = item.Elem()
		}
		row := make([]string, len(cols))
		for j, col := range cols {
			row[j] = formatFieldValue(item.Field(col))
		}
		rows = append(rows, row)
	}

	// 计算每列最大视觉宽度（CJK = 2, ASCII = 1）
	colWidths := make([]int, len(cols))
	for _, row := range rows {
		for j, val := range row {
			if w := visualWidth(val); w > colWidths[j] {
				colWidths[j] = w
			}
		}
	}

	// 输出对齐表格
	for _, row := range rows {
		for j, val := range row {
			if j > 0 {
				fmt.Fprint(w, "  ")
			}
			fmt.Fprint(w, padVisual(val, colWidths[j]))
		}
		fmt.Fprintln(w)
	}
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
	case field.Kind() == reflect.Pointer && !field.IsNil():
		return fmt.Sprintf("%v", field.Elem().Interface())
	case field.Kind() == reflect.Pointer && field.IsNil():
		return "-"
	case field.Kind() == reflect.Slice:
		return fmt.Sprintf("[%d items]", field.Len())
	default:
		return fmt.Sprintf("%v", field.Interface())
	}
}

// visualWidth 计算字符串的终端显示宽度（CJK 字符计为 2，ASCII 计为 1）。
func visualWidth(s string) int {
	w := 0
	for _, r := range s {
		if r > 0x7F {
			w += 2
		} else {
			w += 1
		}
	}
	return w
}

// padVisual 将字符串填充到指定的终端显示宽度。
func padVisual(s string, targetWidth int) string {
	current := visualWidth(s)
	if current >= targetWidth {
		return s
	}
	return s + strings.Repeat(" ", targetWidth-current)
}

// isStructSlice 判断是否为结构体切片（可展开为嵌套表格）。
func isStructSlice(v reflect.Value) bool {
	if v.Kind() != reflect.Slice || v.Len() == 0 {
		return false
	}
	first := v.Index(0)
	if first.Kind() == reflect.Pointer {
		if first.IsNil() {
			return false
		}
		first = first.Elem()
	}
	return first.Kind() == reflect.Struct
}
