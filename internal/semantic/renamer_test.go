package semantic

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRewriteProjectRenamesHashModulesAndUpdatesRequires(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "A3DE0D7433221CDFC5B86573AF447D74.js"), `var e=require("07AF917533221CDF61C9F97247647D74.js");
exports.getECert=function(r){var t={userId:r.userId,controllerName:"CerInfo",methodsName:"GetECert"};return e.request({url:"",method:"GET",data:t})};`)
	mustWrite(t, filepath.Join(root, "07AF917533221CDF61C9F97247647D74.js"), `var t=require("8D7469A633221CDFEB1201A10B047D74.js");
exports.request=function(r){return wx.request({url:t.config.baseApiUrl+r.url,method:r.method,data:r.data})};`)
	mustWrite(t, filepath.Join(root, "8D7469A633221CDFEB1201A10B047D74.js"), `exports.config={baseApiUrl:"https://example.test/api/",baseImgUrl:"https://example.test/img/"};`)
	mustWrite(t, filepath.Join(root, "pages/dzzs/zscontent/content.js"), `var api=require("../../../A3DE0D7433221CDFC5B86573AF447D74.js");api.getECert({userId:"1"});`)

	report, err := RewriteProject(root)
	if err != nil {
		t.Fatalf("RewriteProject 返回错误: %v", err)
	}
	if report.RenamedCount != 3 {
		t.Fatalf("应重命名 3 个模块，got %d", report.RenamedCount)
	}

	assertExists(t, filepath.Join(root, "api_cert.js"))
	assertExists(t, filepath.Join(root, "request.js"))
	assertExists(t, filepath.Join(root, "config.js"))
	assertNotExists(t, filepath.Join(root, "A3DE0D7433221CDFC5B86573AF447D74.js"))

	pageContent := mustRead(t, filepath.Join(root, "pages/dzzs/zscontent/content.js"))
	if !strings.Contains(pageContent, `require("../../../api_cert.js")`) {
		t.Fatalf("页面 require 未被重写: %s", pageContent)
	}

	apiContent := mustRead(t, filepath.Join(root, "api_cert.js"))
	if !strings.Contains(apiContent, "function(params)") || !strings.Contains(apiContent, "var requestData =") {
		t.Fatalf("接口包装变量未语义化: %s", apiContent)
	}
	if !strings.Contains(apiContent, `require("request.js")`) {
		t.Fatalf("根模块 require 未被重写: %s", apiContent)
	}

	requestContent := mustRead(t, filepath.Join(root, "request.js"))
	if !strings.Contains(requestContent, "function(options)") || !strings.Contains(requestContent, "options.url") {
		t.Fatalf("请求包装变量未语义化: %s", requestContent)
	}

	assertExists(t, filepath.Join(root, ".gwxapkg/semantic_module_map.json"))

	secondReport, err := RewriteProject(root)
	if err != nil {
		t.Fatalf("第二次 RewriteProject 返回错误: %v", err)
	}
	if secondReport.RenamedCount != report.RenamedCount {
		t.Fatalf("重复执行不应覆盖已有映射，got %d want %d", secondReport.RenamedCount, report.RenamedCount)
	}
}

func TestRewriteProjectRecoversSourcesContent(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "app.js"), "console.log(1);\n//# sourceMappingURL=app.js.map\n")
	mustWrite(t, filepath.Join(root, "app.js.map"), `{"version":3,"sources":["webpack://src/pages/index/index.js"],"sourcesContent":["Page({data:{}})"]}`)

	report, err := RewriteProject(root)
	if err != nil {
		t.Fatalf("RewriteProject 返回错误: %v", err)
	}
	if report.SourceMapRecovered != 1 {
		t.Fatalf("应恢复 1 个 source map 源文件，got %d", report.SourceMapRecovered)
	}
	assertExists(t, filepath.Join(root, ".gwxapkg/sources/app.js/src/pages/index/index.js"))
}

func TestBuildAPIMapSplitsMixedControllerModule(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "api_mixed.js"), `var request=require("request.js");
exports.getECert=function(params){var requestData={userId:params.userId,controllerName:"CerInfo",methodsName:"GetECert"};return request.request({url:"",method:"GET",data:requestData})};
exports.getDriverLicense=function(params){var requestData={userId:params.userId,controllerName:"DrivingLicense",methodsName:"GetImageURL"};return request.request({url:"",method:"GET",data:requestData})};`)
	mustWrite(t, filepath.Join(root, "request.js"), `exports.request=function(options){return wx.request(options)};`)
	mustWrite(t, filepath.Join(root, "pages/index/index.js"), `var api=require("../../api_mixed.js");api.getECert({userId:"1"});api.getDriverLicense({userId:"1"});`)

	report, err := BuildAPIMap(root, []string{
		"api_mixed.js",
		"request.js",
		"pages/index/index.js",
	})
	if err != nil {
		t.Fatalf("BuildAPIMap 返回错误: %v", err)
	}
	if report.SplitModuleCount != 1 {
		t.Fatalf("应细拆 1 个 API 模块，got %d", report.SplitModuleCount)
	}
	if len(report.Endpoints) != 2 {
		t.Fatalf("应输出 2 个接口，got %d", len(report.Endpoints))
	}

	assertExists(t, filepath.Join(root, "api/cert.js"))
	assertExists(t, filepath.Join(root, "api/driving_license.js"))
	assertExists(t, filepath.Join(root, ".gwxapkg/api_map.json"))
	assertExists(t, filepath.Join(root, ".gwxapkg/api_map.md"))

	barrel := mustRead(t, filepath.Join(root, "api_mixed.js"))
	if !strings.Contains(barrel, `require("./api/cert.js")`) || !strings.Contains(barrel, `exports.getECert`) {
		t.Fatalf("兼容入口内容不正确: %s", barrel)
	}

	foundCallSite := false
	for _, endpoint := range report.Endpoints {
		if endpoint.FunctionName == "getECert" && len(endpoint.CallSites) == 1 && endpoint.CallSites[0].FilePath == "pages/index/index.js" {
			foundCallSite = true
		}
	}
	if !foundCallSite {
		t.Fatalf("api_map 应记录页面调用点: %#v", report.Endpoints)
	}
}

func mustWrite(t *testing.T, filename string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("写文件失败: %v", err)
	}
}

func mustRead(t *testing.T, filename string) string {
	t.Helper()
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("读文件失败: %v", err)
	}
	return string(data)
}

func assertExists(t *testing.T, filename string) {
	t.Helper()
	if _, err := os.Stat(filename); err != nil {
		t.Fatalf("期望文件存在 %s: %v", filename, err)
	}
}

func assertNotExists(t *testing.T, filename string) {
	t.Helper()
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		t.Fatalf("期望文件不存在 %s: %v", filename, err)
	}
}
