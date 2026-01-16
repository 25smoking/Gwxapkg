package unpack

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/dop251/goja"

	"github.com/25smoking/Gwxapkg/internal/enum"

	"github.com/25smoking/Gwxapkg/internal/config"
)

// JavaScriptParser JavaScript 解析器
type JavaScriptParser struct {
	OutputDir string
}

// DefineParams 存储从 define 函数中提取的参数
type DefineParams struct {
	ModuleName string
	FuncBody   string
}

func cleanDefineFunc(jsCode string) string {
	// 正则表达式匹配 define 函数的头部
	reHead := regexp.MustCompile(`^define\s*\(\s*["'].*?["']\s*,\s*function\s*\([^)]*\)\s*\{`)

	// 正则表达式匹配 define 函数的尾部
	reTail := regexp.MustCompile(`\}\s*,\s*\{[^}]*isPage\s*:\s*[^}]*\}\s*\)\s*;$`)

	// 移除头部
	cleanedCode := reHead.ReplaceAllString(jsCode, "")

	// 移除尾部
	cleanedCode = reTail.ReplaceAllString(cleanedCode, "")

	// 去除开头和结尾的空白字符
	cleanedCode = strings.TrimSpace(cleanedCode)

	// 去除"严格模式"声明
	if strings.HasPrefix(cleanedCode, `"use strict";`) || strings.HasPrefix(cleanedCode, `'use strict';`) {
		cleanedCode = cleanedCode[13:]
		cleanedCode = strings.TrimSpace(cleanedCode)
	}

	return cleanedCode
}

// extractDefineParams 提取所有 define 函数的第一个和第二个参数
func extractDefineParams(jsCode string) ([]DefineParams, error) {
	// 正则表达式提取 define 函数的第一个和第二个参数 (严格匹配)
	re := regexp.MustCompile(`define\s*\(\s*["']([^"']+)["']\s*,\s*function\s*\(([^)]*)\)\s*\{([\s\S]*?)\}\s*,\s*\{[^}]*isPage\s*:\s*[^}]*\}\s*\)\s*;`)
	matches := re.FindAllStringSubmatch(jsCode, -1)

	if len(matches) == 0 {
		// 尝试使用 goja 运行 JavaScript
		results, err := run(jsCode)
		if err != nil {
			// 如果 goja 失败 (例如 SyntaxError)，使用备选的宽松正则表达式
			log.Printf("警告: goja 执行失败 (%v), 尝试使用正则表达式解析", err)
			fallbackResults := extractDefineParamsFallback(jsCode)
			if len(fallbackResults) > 0 {
				return fallbackResults, nil
			}
			// 如果备选方案也失败，返回空结果而非错误
			return []DefineParams{}, nil
		}
		return results, nil
	}

	var results = make([]DefineParams, 0)
	for _, match := range matches {
		if len(match) >= 3 {
			params := DefineParams{
				ModuleName: match[1],
				FuncBody:   cleanDefineFunc(match[0]),
			}
			results = append(results, params)
		}
	}
	return results, nil
}

// extractDefineParamsFallback 使用宽松的正则表达式作为备选方案
func extractDefineParamsFallback(jsCode string) []DefineParams {
	var results = make([]DefineParams, 0)

	// 更宽松的正则表达式，匹配 define("path", function(...) {...})
	re := regexp.MustCompile(`define\s*\(\s*["']([^"']+\.js)["']\s*,\s*function\s*\([^)]*\)\s*\{`)

	// 查找所有 define 调用的起始位置
	allMatches := re.FindAllStringSubmatchIndex(jsCode, -1)

	for _, loc := range allMatches {
		if len(loc) >= 4 {
			moduleName := jsCode[loc[2]:loc[3]]
			startIdx := loc[1] // define 函数体开始位置

			// 使用括号匹配找到函数体结束位置
			funcBody := extractFunctionBody(jsCode, startIdx-1)
			if funcBody != "" {
				params := DefineParams{
					ModuleName: moduleName,
					FuncBody:   strings.TrimSpace(funcBody),
				}
				results = append(results, params)
			}
		}
	}

	if len(results) > 0 {
		log.Printf("使用正则表达式成功提取了 %d 个模块", len(results))
	}

	return results
}

// extractFunctionBody 从给定位置提取函数体内容（使用括号匹配）
func extractFunctionBody(code string, startIdx int) string {
	if startIdx < 0 || startIdx >= len(code) {
		return ""
	}

	braceCount := 1
	endIdx := startIdx + 1

	for endIdx < len(code) && braceCount > 0 {
		ch := code[endIdx]
		if ch == '{' {
			braceCount++
		} else if ch == '}' {
			braceCount--
		}
		endIdx++
	}

	if braceCount == 0 && endIdx > startIdx+1 {
		// 返回函数体内容（不包括外层大括号）
		body := code[startIdx+1 : endIdx-1]
		// 清理 "use strict" 声明
		body = strings.TrimSpace(body)
		if strings.HasPrefix(body, `"use strict";`) || strings.HasPrefix(body, `'use strict';`) {
			body = strings.TrimSpace(body[13:])
		}
		return body
	}

	return ""
}

func run(code string) ([]DefineParams, error) {
	var results = make([]DefineParams, 0)
	var jsError string // 用于捕获 JavaScript 错误信息

	// 添加更多全局变量 mock 以防止 ReferenceError
	patch := `var window={};var navigator={};navigator.userAgent="iPhone";window.screen={};
document={getElementsByTagName:()=>{},createElement:()=>({}),body:{},head:{}};
function require(){return {}};var global={};var __wxAppCode__={};var __wxConfig={};
var __vd_version_info__={};var $gwx=function(){return function(){}};var __g={};
var WeixinJSBridge={invoke:()=>{},on:()=>{},publish:()=>{},subscribe:()=>{}};
var wx={};var getApp=function(){return {}};var getCurrentPages=function(){return []};
var App=function(){};var Page=function(){};var Component=function(){};var Behavior=function(){};
var getRegExp=function(){return new RegExp()};var getDate=function(){return new Date()};
var console={log:function(){},error:function(){},warn:function(){},info:function(){}};
var setTimeout=function(){};var setInterval=function(){};var clearTimeout=function(){};var clearInterval=function(){};
`

	scriptcode := patch + code

	// 包裹 try...catch 语句以捕获 JavaScript 错误
	safeScript := `
	try {
		` + string(scriptcode) + `
	} catch (e) {
		__jsError__ = e.toString();
	}
	`

	vm := goja.New()

	// 定义 console 对象
	console := vm.NewObject()
	_ = console.Set("log", func(call goja.FunctionCall) goja.Value {
		// 使用 call.Arguments 获取传递给 console.log 的参数
		args := call.Arguments
		for _, arg := range args {
			fmt.Println(arg.String())
		}
		return goja.Undefined()
	})
	_ = console.Set("error", func(call goja.FunctionCall) goja.Value {
		args := call.Arguments
		for _, arg := range args {
			fmt.Println("ERROR:", arg.String())
		}
		return goja.Undefined()
	})
	_ = console.Set("warn", func(call goja.FunctionCall) goja.Value {
		args := call.Arguments
		for _, arg := range args {
			fmt.Println("WARN:", arg.String())
		}
		return goja.Undefined()
	})
	_ = vm.Set("console", console)

	// 设置 define 函数和 require 函数的行为
	err := vm.Set("define", func(call goja.FunctionCall) goja.Value {
		moduleName := call.Argument(0).String()
		funcBody := call.Argument(1).String()

		cleanedCode, err := removeWrapper(funcBody)
		if err != nil {
			log.Printf("Error removing wrapper: %v\n", err)
			cleanedCode = funcBody
		}

		//检查是否包含 "use strict" 并处理
		if strings.HasPrefix(cleanedCode, `"use strict";`) || strings.HasPrefix(cleanedCode, `'use strict';`) {
			cleanedCode = cleanedCode[13:]
		} else if (strings.HasPrefix(cleanedCode, `(function(){"use strict";`) || strings.HasPrefix(cleanedCode, `(function(){'use strict';`)) &&
			strings.HasSuffix(cleanedCode, `})();`) {
			cleanedCode = cleanedCode[25 : len(cleanedCode)-5]
		}

		params := DefineParams{
			ModuleName: moduleName,
			FuncBody:   cleanedCode,
		}
		results = append(results, params)

		return goja.Undefined()
	})
	if err != nil {
		return nil, err
	}

	// 初始化错误捕获变量
	_ = vm.Set("__jsError__", "")

	_, err = vm.RunString(safeScript)
	if err != nil {
		return nil, fmt.Errorf("failed to run JavaScript: %w", err)
	}

	// 检查 JavaScript 运行时错误
	if jsErrorVal := vm.Get("__jsError__"); jsErrorVal != nil && !goja.IsUndefined(jsErrorVal) && !goja.IsNull(jsErrorVal) {
		jsError = jsErrorVal.String()
		if jsError != "" {
			log.Printf("警告: JavaScript 执行时发生错误: %s", jsError)
		}
	}

	// 如果没有提取到任何模块，记录警告
	if len(results) == 0 {
		log.Printf("警告: 未能从 JavaScript 文件中提取任何模块定义")
	}

	return results, nil
}

// removeWrapper 移除函数包装器
func removeWrapper(jsCode string) (string, error) {
	vm := goja.New()
	script := `
		(function(code) {
			let match = code.match(/^function\s*\(.*?\)\s*\{([\s\S]*)\}$/);
			if (match && match[1]) {
				// 每一行缩进减少一个空格
				match[1] = match[1].trim();
				code = match[1].replace(/^\s{4}/gm, '');
			}
			return code;
		})(code);
	`
	// 设置 JavaScript 变量
	err := vm.Set("code", jsCode)
	if err != nil {
		return "", err
	}
	value, err := vm.RunString(script)
	if err != nil {
		return "", fmt.Errorf("JavaScript execution error: %w", err)
	}
	return value.String(), nil
}

// 是否为分包
func isSubpackage(wxapkg *config.WxapkgInfo) bool {
	switch wxapkg.WxapkgType {
	case enum.APP_SUBPACKAGE_V1, enum.APP_SUBPACKAGE_V2, enum.GAME_SUBPACKAGE:
		return true
	default:
		return false
	}
}

// Parse 解析和分割 JavaScript 文件
func (p *JavaScriptParser) Parse(option config.WxapkgInfo) error {

	dir := option.SourcePath
	if isSubpackage(&option) {
		dir = p.OutputDir
	}

	code, err := os.ReadFile(option.Option.ServiceSource)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	params, err := extractDefineParams(string(code))
	if err != nil {
		return err
	}

	// 并行保存文件
	var wg sync.WaitGroup
	sem := make(chan struct{}, 20) // 限制并发数为20

	for _, param := range params {
		wg.Add(1)
		sem <- struct{}{} // 获取信号量
		go func(param DefineParams) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量
			err := save(filepath.Join(dir, param.ModuleName), []byte(param.FuncBody))
			if err != nil {
				log.Printf("Error saving file: %v\n", err)
			}
		}(param)
	}

	wg.Wait()
	// log.Printf("Splitting \"%s\" done.", option.Option.ServiceSource)
	return nil
}
