// Package eastmoney 提供东方财富自动交易接口的 Go 语言实现。
//
// 使用示例（本地 OCR）：
//
//	recognizer, err := captcha.NewDefaultRecognizer(modelPath, dictPath, onnxLibPath)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer recognizer.Close()
//
//	c, err := client.NewClient("username", "password", recognizer)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	resp, err := c.QueryAssetAndPosition()
//
// 使用示例（远程 OCR）：
//
//	recognizer := captcha.NewRemoteRecognizer("http://localhost:8080/ocr")
//	c, err := client.NewClient("username", "password", recognizer)
package eastmoney
