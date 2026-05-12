package semantic

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRenameIdentifiersRespectsScopeAndClosure(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "pages/index/index.js"), `function outer(e){
var t=getApp();
function inner(){return e.userId+t.globalData.token}
function shadow(e){return e.detail.value}
return inner()+shadow({detail:{value:1}})
}`)

	report, err := RenameIdentifiersWithOptions(root, []string{"pages/index/index.js"}, ASTRenameOptions{
		Mode: ASTRenameModeDeep,
	})
	if err != nil {
		t.Fatalf("RenameIdentifiers 返回错误: %v", err)
	}
	if report.TotalRenames == 0 {
		t.Fatalf("应产生 AST 重命名: %#v", report)
	}

	content := mustRead(t, filepath.Join(root, "pages/index/index.js"))
	if !strings.Contains(content, "function outer(params)") || !strings.Contains(content, "params.userId") {
		t.Fatalf("外层参数及闭包引用应同步改名: %s", content)
	}
	if !strings.Contains(content, "var app=getApp()") || !strings.Contains(content, "app.globalData.token") {
		t.Fatalf("getApp 别名及闭包引用应同步改名: %s", content)
	}
	if !strings.Contains(content, "function shadow(event)") || !strings.Contains(content, "event.detail.value") {
		t.Fatalf("内层 shadow 参数应独立改名: %s", content)
	}
}

func TestRenameIdentifiersSkipsObjectPropertiesStringsAndExports(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "api/cert.js"), `exports.getECert=function(e){
var t={userId:e.userId,controllerName:"CerInfo",methodsName:"GetECert"};
var n="e should stay";
return request.request({data:t})
};
Page({onLoad:function(e){this.setData({userId:e.userId})}});`)

	_, err := RenameIdentifiersWithOptions(root, []string{"api/cert.js"}, ASTRenameOptions{
		Mode: ASTRenameModeDeep,
	})
	if err != nil {
		t.Fatalf("RenameIdentifiers 返回错误: %v", err)
	}

	content := mustRead(t, filepath.Join(root, "api/cert.js"))
	for _, mustKeep := range []string{
		"exports.getECert",
		"controllerName",
		"methodsName",
		"userId:",
		`"e should stay"`,
		"onLoad:function",
	} {
		if !strings.Contains(content, mustKeep) {
			t.Fatalf("不应破坏公开导出、业务字段、字符串或页面 handler: want %q in %s", mustKeep, content)
		}
	}
	if !strings.Contains(content, "function(params)") || !strings.Contains(content, "var requestData=") {
		t.Fatalf("接口参数和请求体变量应被语义化: %s", content)
	}
}

func TestRenameIdentifiersRenamesRequireAlias(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "pages/index/index.js"), `var e=require("../../api/cert.js");
e.getECert({userId:"1"});`)

	_, err := RenameIdentifiersWithOptions(root, []string{"pages/index/index.js"}, ASTRenameOptions{
		Mode: ASTRenameModeSafe,
	})
	if err != nil {
		t.Fatalf("RenameIdentifiers 返回错误: %v", err)
	}

	content := mustRead(t, filepath.Join(root, "pages/index/index.js"))
	if !strings.Contains(content, `var apiCert=require("../../api/cert.js")`) || !strings.Contains(content, "apiCert.getECert") {
		t.Fatalf("require 别名应按语义模块名改名: %s", content)
	}
}

func TestRenameIdentifiersDoesNotMisnameCallbackLocalAsApp(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "pages/index/index.js"), `function setup(){
var t=getApp();
onLoad(function(n){
var r=n.userid;
var a=t.globalData.userInfo.userID;
return r || a;
});
}`)

	_, err := RenameIdentifiersWithOptions(root, []string{"pages/index/index.js"}, ASTRenameOptions{
		Mode: ASTRenameModeDeep,
	})
	if err != nil {
		t.Fatalf("RenameIdentifiers 返回错误: %v", err)
	}

	content := mustRead(t, filepath.Join(root, "pages/index/index.js"))
	if !strings.Contains(content, "var app=getApp()") || !strings.Contains(content, "app.globalData.userInfo.userID") {
		t.Fatalf("getApp 别名应保持可读且闭包引用同步: %s", content)
	}
	if strings.Contains(content, "var app=n.userid") {
		t.Fatalf("回调局部变量不能误命名为 app 并遮蔽外层 app: %s", content)
	}
}

func TestRenameIdentifiersModesConfidenceAndRollback(t *testing.T) {
	root := t.TempDir()
	rel := "pages/index/index.js"
	original := `var e=require("../../api/cert.js");
function onTap(t){return t.detail.value}
e.getECert({userId:"1"});`
	mustWrite(t, filepath.Join(root, filepath.FromSlash(rel)), original)

	report, err := RenameIdentifiersWithOptions(root, []string{rel}, ASTRenameOptions{
		Mode:          ASTRenameModeSafe,
		GenerateDiff:  true,
		GeneratePatch: true,
	})
	if err != nil {
		t.Fatalf("safe 模式返回错误: %v", err)
	}
	content := mustRead(t, filepath.Join(root, filepath.FromSlash(rel)))
	if !strings.Contains(content, "apiCert.getECert") {
		t.Fatalf("safe 应写回 high confidence require: %s", content)
	}
	if strings.Contains(content, "function onTap(event)") {
		t.Fatalf("safe 不应写回 medium confidence event: %s", content)
	}
	foundMediumReportOnly := false
	for _, file := range report.Files {
		for _, rename := range file.Renames {
			if rename.NewName == "event" && rename.Confidence == ASTConfidenceMedium && !rename.Applied {
				foundMediumReportOnly = true
			}
		}
	}
	if !foundMediumReportOnly {
		t.Fatalf("safe 应报告但不写回 medium: %#v", report.Files)
	}
	assertExists(t, filepath.Join(root, ".gwxapkg/ast_rename_diff.md"))
	assertExists(t, filepath.Join(root, ".gwxapkg/ast_rename.patch"))

	rollback, err := RollbackASTRenames(root)
	if err != nil {
		t.Fatalf("回滚失败: %v", err)
	}
	if len(rollback.RestoredFiles) != 1 {
		t.Fatalf("应回滚 1 个文件: %#v", rollback)
	}
	if got := mustRead(t, filepath.Join(root, filepath.FromSlash(rel))); got != original {
		t.Fatalf("回滚后内容不一致:\n%s", got)
	}

	_, err = RenameIdentifiersWithOptions(root, []string{rel}, ASTRenameOptions{Mode: ASTRenameModeDeep})
	if err != nil {
		t.Fatalf("deep 模式返回错误: %v", err)
	}
	content = mustRead(t, filepath.Join(root, filepath.FromSlash(rel)))
	if !strings.Contains(content, "function onTap(event)") {
		t.Fatalf("deep 应写回 medium confidence event: %s", content)
	}
}

func TestRenameIdentifiersReportsParseSkip(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "broken.js"), `function(){`)

	report, err := RenameIdentifiers(root, []string{"broken.js"})
	if err != nil {
		t.Fatalf("解析失败文件不应阻断流程: %v", err)
	}
	if len(report.Files) != 1 || report.Files[0].Status != "skipped" || report.Files[0].Error == "" {
		t.Fatalf("解析失败应被记录为 skipped: %#v", report.Files)
	}
}
