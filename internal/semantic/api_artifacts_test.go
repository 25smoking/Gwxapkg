package semantic

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildAPIMapWritesCallChainAndPseudo(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "api_mixed.js"), `var request=require("request.js");
exports.getECert=function(params){var requestData={userId:params.userId,controllerName:"CerInfo",methodsName:"GetECert"};return request.request({url:"",method:"GET",data:requestData})};`)
	mustWrite(t, filepath.Join(root, "request.js"), `exports.request=function(options){return wx.request(options)};`)
	mustWrite(t, filepath.Join(root, "pages/index/index.js"), `var api=require("../../api_mixed.js");
function loadCert(){return api.getECert({userId:"1"})}
Page({onLoad:function(){loadCert()}});`)

	report, err := BuildAPIMap(root, []string{"api_mixed.js", "request.js", "pages/index/index.js"})
	if err != nil {
		t.Fatalf("BuildAPIMap 返回错误: %v", err)
	}
	if len(report.CallChains) != 1 {
		t.Fatalf("应生成 1 条 API 调用链: %#v", report.CallChains)
	}
	if report.CallChains[0].APIFunction != "getECert" || report.CallChains[0].ControllerName != "CerInfo" {
		t.Fatalf("调用链应指向 getECert/CerInfo: %#v", report.CallChains[0])
	}
	assertExists(t, filepath.Join(root, ".gwxapkg/api_call_chain.json"))
	assertExists(t, filepath.Join(root, ".gwxapkg/api_call_chain.md"))
	assertExists(t, filepath.Join(root, ".gwxapkg/api_pseudo.md"))
	pseudo := mustRead(t, filepath.Join(root, ".gwxapkg/pseudo_api/cert.js"))
	if !strings.Contains(pseudo, "function getECert({ userId })") ||
		!strings.Contains(pseudo, `controllerName: "CerInfo"`) ||
		!strings.Contains(pseudo, `methodsName: "GetECert"`) {
		t.Fatalf("伪代码内容不正确: %s", pseudo)
	}
}

func TestLinkBurpRequestMatchesAPIMap(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "api_mixed.js"), `var request=require("request.js");
exports.getECert=function(params){var requestData={userId:params.userId,controllerName:"CerInfo",methodsName:"GetECert"};return request.request({url:"",method:"GET",data:requestData})};`)
	mustWrite(t, filepath.Join(root, "request.js"), `exports.request=function(options){return wx.request(options)};`)
	mustWrite(t, filepath.Join(root, "pages/index/index.js"), `var api=require("../../api_mixed.js");api.getECert({userId:"1"});`)
	if _, err := BuildAPIMap(root, []string{"api_mixed.js", "request.js", "pages/index/index.js"}); err != nil {
		t.Fatalf("BuildAPIMap 返回错误: %v", err)
	}

	raw := "POST /gateway HTTP/1.1\r\nHost: example.test\r\nContent-Type: application/json\r\n\r\n{\"controllerName\":\"CerInfo\",\"methodsName\":\"GetECert\",\"userId\":\"1\"}"
	report, err := LinkBurpRequest(root, raw)
	if err != nil {
		t.Fatalf("LinkBurpRequest 返回错误: %v", err)
	}
	if len(report.Matches) == 0 || report.Matches[0].Confidence != ASTConfidenceHigh || report.Matches[0].FunctionName != "getECert" {
		t.Fatalf("Burp 请求应 high 匹配 getECert: %#v", report.Matches)
	}
	assertExists(t, filepath.Join(root, ".gwxapkg/burp_api_link.json"))
	assertExists(t, filepath.Join(root, ".gwxapkg/burp_api_link.md"))
}
