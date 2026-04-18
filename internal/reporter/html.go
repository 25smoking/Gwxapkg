package reporter

import (
	"fmt"
	"html/template"
	"os"
	"sort"
	"strings"

	"github.com/25smoking/Gwxapkg/internal/scanner"
)

// HTMLReporter HTML 报告生成器
type HTMLReporter struct{}

// NewHTMLReporter 创建 HTML 报告生成器
func NewHTMLReporter() *HTMLReporter {
	return &HTMLReporter{}
}

// Generate 生成 HTML 报告
func (r *HTMLReporter) Generate(report *scanner.ScanReport, filename string) error {
	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"pct": func(part, total int) string {
			if total == 0 {
				return "0.0"
			}
			return fmt.Sprintf("%.1f", float64(part)/float64(total)*100)
		},
		"riskClass": func(confidence string) string {
			switch confidence {
			case "high":
				return "risk-high"
			case "medium":
				return "risk-med"
			default:
				return "risk-low"
			}
		},
		"riskLabel": func(confidence string) string {
			switch confidence {
			case "high":
				return "高"
			case "medium":
				return "中"
			default:
				return "低"
			}
		},
	}).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("解析模板失败: %w", err)
	}

	// 构建渲染数据
	data := buildHTMLData(report)

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("渲染模板失败: %w", err)
	}
	return nil
}

// HTMLData 模板数据
type HTMLData struct {
	AppID        string
	ScanTime     string
	TotalFiles   int
	TotalMatches int
	UniqueCount  int
	HighRisk     int
	MediumRisk   int
	LowRisk      int
	Categories   []HTMLCategory
	AllItems     []HTMLItem
}

type HTMLCategory struct {
	Key   string
	Name  string
	Count int
	Items []HTMLItem
}

type HTMLItem struct {
	Content    string
	Count      int
	FilePath   string
	LineNumber int
	Context    string
	Confidence string
	Category   string
}

func buildHTMLData(report *scanner.ScanReport) HTMLData {
	data := HTMLData{
		AppID:        report.AppID,
		ScanTime:     report.ScanTime,
		TotalFiles:   report.TotalFiles,
		TotalMatches: report.Summary.TotalMatches,
		UniqueCount:  report.Summary.UniqueMatches,
		HighRisk:     report.Summary.HighRisk,
		MediumRisk:   report.Summary.MediumRisk,
		LowRisk:      report.Summary.LowRisk,
	}

	// 构建分类数据
	var catKeys []string
	for k := range report.Categories {
		catKeys = append(catKeys, k)
	}
	sort.Strings(catKeys)

	// 构建 content -> confidence 映射（从 Items）
	confidenceMap := make(map[string]string)
	contextMap := make(map[string]string)
	catMap := make(map[string]string)
	for _, item := range report.Items {
		key := item.RuleID + ":" + item.Content
		confidenceMap[key] = item.Confidence
		contextMap[key] = strings.TrimSpace(item.Context)
		catMap[item.Content] = item.Category
	}

	for _, k := range catKeys {
		catData := report.Categories[k]
		cat := HTMLCategory{
			Key:   k,
			Name:  catData.Name,
			Count: catData.UniqueCount,
		}

		var contents []string
		for content := range catData.Items {
			contents = append(contents, content)
		}
		sort.Strings(contents)

		for _, content := range contents {
			locs := catData.Items[content]
			fp := ""
			ln := 0
			if len(locs) > 0 {
				fp = locs[0].FilePath
				ln = locs[0].LineNumber
			}
			conf := confidenceMap[k+":"+content]
			if conf == "" {
				conf = "low"
			}
			item := HTMLItem{
				Content:    content,
				Count:      len(locs),
				FilePath:   fp,
				LineNumber: ln,
				Context:    contextMap[k+":"+content],
				Confidence: conf,
				Category:   catData.Name,
			}
			cat.Items = append(cat.Items, item)
			data.AllItems = append(data.AllItems, item)
		}
		data.Categories = append(data.Categories, cat)
	}

	return data
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Gwxapkg 敏感信息报告 - {{.AppID}}</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:#0f1117;color:#e1e4e8;min-height:100vh}
.header{background:linear-gradient(135deg,#1a1d27 0%,#12151e 100%);border-bottom:1px solid #21262d;padding:24px 32px}
.header h1{font-size:22px;font-weight:700;color:#58a6ff;letter-spacing:.5px}
.header .meta{margin-top:8px;font-size:13px;color:#8b949e;display:flex;gap:24px;flex-wrap:wrap}
.header .meta span{display:flex;align-items:center;gap:6px}
.container{max-width:1400px;margin:0 auto;padding:24px 32px}
.stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:16px;margin-bottom:28px}
.stat-card{background:#161b22;border:1px solid #21262d;border-radius:10px;padding:18px 20px;transition:border-color .2s}
.stat-card:hover{border-color:#30363d}
.stat-card .val{font-size:32px;font-weight:700;line-height:1}
.stat-card .lbl{font-size:12px;color:#8b949e;margin-top:6px}
.stat-card.high .val{color:#f85149}
.stat-card.med .val{color:#e3b341}
.stat-card.low .val{color:#3fb950}
.stat-card.total .val{color:#58a6ff}
.stat-card.unique .val{color:#bc8cff}
.risk-bar{background:#161b22;border:1px solid #21262d;border-radius:10px;padding:18px 20px;margin-bottom:28px}
.risk-bar h3{font-size:13px;color:#8b949e;margin-bottom:12px}
.bar-track{height:12px;border-radius:6px;background:#21262d;overflow:hidden;display:flex}
.bar-seg{height:100%;transition:width .4s}
.bar-seg.h{background:#f85149}
.bar-seg.m{background:#e3b341}
.bar-seg.l{background:#3fb950}
.bar-labels{display:flex;gap:20px;margin-top:8px;font-size:12px}
.bar-labels span{display:flex;align-items:center;gap:5px}
.bar-labels .dot{width:10px;height:10px;border-radius:50%}
.search-wrap{position:relative;margin-bottom:20px}
.search-wrap input{width:100%;background:#161b22;border:1px solid #30363d;border-radius:8px;padding:10px 16px 10px 40px;color:#e1e4e8;font-size:14px;outline:none;transition:border-color .2s}
.search-wrap input:focus{border-color:#58a6ff}
.search-wrap .ico{position:absolute;left:14px;top:50%;transform:translateY(-50%);color:#555;font-size:14px}
.tabs{display:flex;gap:4px;flex-wrap:wrap;margin-bottom:20px;background:#0d1117;border:1px solid #21262d;border-radius:10px;padding:6px}
.tab{padding:7px 14px;border-radius:7px;cursor:pointer;font-size:13px;font-weight:500;color:#8b949e;transition:all .15s;white-space:nowrap}
.tab:hover{color:#e1e4e8;background:#21262d}
.tab.active{background:#1f6feb;color:#fff}
.tab .badge{background:#30363d;border-radius:10px;padding:1px 7px;font-size:11px;margin-left:5px}
.tab.active .badge{background:rgba(255,255,255,.2)}
.panel{display:none}
.panel.active{display:block}
.table-wrap{overflow-x:auto;border:1px solid #21262d;border-radius:10px}
table{width:100%;border-collapse:collapse;font-size:13px}
thead th{background:#161b22;padding:11px 14px;text-align:left;font-weight:600;color:#8b949e;font-size:12px;text-transform:uppercase;letter-spacing:.4px;white-space:nowrap;border-bottom:1px solid #21262d}
tbody tr{border-bottom:1px solid #0d1117;transition:background .1s}
tbody tr:hover{background:#161b22}
tbody tr:last-child{border-bottom:none}
td{padding:10px 14px;vertical-align:top}
td.content-cell{max-width:320px;word-break:break-all;font-family:monospace;font-size:12px;color:#79c0ff}
td.path-cell{max-width:260px;word-break:break-all;color:#8b949e;font-size:12px}
td.ctx-cell{max-width:360px;word-break:break-all;color:#6e7681;font-size:11px;font-family:monospace}
.risk-badge{display:inline-block;padding:2px 8px;border-radius:4px;font-size:11px;font-weight:600}
.risk-high{background:rgba(248,81,73,.15);color:#f85149;border:1px solid rgba(248,81,73,.3)}
.risk-med{background:rgba(227,179,65,.12);color:#e3b341;border:1px solid rgba(227,179,65,.3)}
.risk-low{background:rgba(63,185,80,.12);color:#3fb950;border:1px solid rgba(63,185,80,.3)}
.no-data{text-align:center;padding:48px;color:#484f58}
.no-data .ico{font-size:40px;margin-bottom:12px}
.footer{text-align:center;padding:24px;color:#484f58;font-size:12px;border-top:1px solid #21262d;margin-top:24px}
</style>
</head>
<body>
<div class="header">
  <h1>🔍 Gwxapkg 敏感信息扫描报告</h1>
  <div class="meta">
    <span>📦 AppID: <b style="color:#e1e4e8">{{.AppID}}</b></span>
    <span>🕐 扫描时间: {{.ScanTime}}</span>
    <span>📄 扫描文件: {{.TotalFiles}} 个</span>
  </div>
</div>
<div class="container">

<div class="stats">
  <div class="stat-card total"><div class="val">{{.TotalMatches}}</div><div class="lbl">总匹配数</div></div>
  <div class="stat-card unique"><div class="val">{{.UniqueCount}}</div><div class="lbl">去重后数量</div></div>
  <div class="stat-card high"><div class="val">{{.HighRisk}}</div><div class="lbl">🔴 高风险</div></div>
  <div class="stat-card med"><div class="val">{{.MediumRisk}}</div><div class="lbl">🟡 中风险</div></div>
  <div class="stat-card low"><div class="val">{{.LowRisk}}</div><div class="lbl">🟢 低风险</div></div>
</div>

{{if gt .UniqueCount 0}}
<div class="risk-bar">
  <h3>风险分布</h3>
  <div class="bar-track">
    <div class="bar-seg h" style="width:{{pct .HighRisk .UniqueCount}}%"></div>
    <div class="bar-seg m" style="width:{{pct .MediumRisk .UniqueCount}}%"></div>
    <div class="bar-seg l" style="width:{{pct .LowRisk .UniqueCount}}%"></div>
  </div>
  <div class="bar-labels">
    <span><span class="dot" style="background:#f85149"></span>高风险 {{pct .HighRisk .UniqueCount}}%</span>
    <span><span class="dot" style="background:#e3b341"></span>中风险 {{pct .MediumRisk .UniqueCount}}%</span>
    <span><span class="dot" style="background:#3fb950"></span>低风险 {{pct .LowRisk .UniqueCount}}%</span>
  </div>
</div>
{{end}}

<div class="search-wrap">
  <span class="ico">🔎</span>
  <input type="text" id="search" placeholder="搜索内容、路径、上下文..." oninput="filterTable()">
</div>

<div class="tabs" id="tabs">
  <div class="tab active" onclick="switchTab('all',this)">全部<span class="badge">{{.UniqueCount}}</span></div>
  {{range .Categories}}
  <div class="tab" onclick="switchTab('{{.Key}}',this)">{{.Name}}<span class="badge">{{.Count}}</span></div>
  {{end}}
</div>

<!-- 全部 -->
<div class="panel active" id="panel-all">
  {{if eq (len .AllItems) 0}}
  <div class="no-data"><div class="ico">✅</div><div>未发现敏感信息</div></div>
  {{else}}
  <div class="table-wrap">
  <table id="tbl-all">
    <thead><tr><th>#</th><th>内容</th><th>分类</th><th>风险</th><th>出现次数</th><th>文件路径</th><th>行号</th><th>上下文</th></tr></thead>
    <tbody>
    {{range $i,$item := .AllItems}}
    <tr>
      <td style="color:#484f58;white-space:nowrap">{{add $i 1}}</td>
      <td class="content-cell">{{$item.Content}}</td>
      <td style="white-space:nowrap;color:#8b949e">{{$item.Category}}</td>
      <td><span class="risk-badge {{riskClass $item.Confidence}}">{{riskLabel $item.Confidence}}</span></td>
      <td style="text-align:center;color:#8b949e">{{$item.Count}}</td>
      <td class="path-cell">{{$item.FilePath}}</td>
      <td style="text-align:center;color:#8b949e">{{$item.LineNumber}}</td>
      <td class="ctx-cell">{{$item.Context}}</td>
    </tr>
    {{end}}
    </tbody>
  </table>
  </div>
  {{end}}
</div>

<!-- 分类面板 -->
{{range .Categories}}
<div class="panel" id="panel-{{.Key}}">
  {{if eq (len .Items) 0}}
  <div class="no-data"><div class="ico">✅</div><div>该分类无数据</div></div>
  {{else}}
  <div class="table-wrap">
  <table id="tbl-{{.Key}}">
    <thead><tr><th>#</th><th>内容</th><th>风险</th><th>出现次数</th><th>文件路径</th><th>行号</th><th>上下文</th></tr></thead>
    <tbody>
    {{range $i,$item := .Items}}
    <tr>
      <td style="color:#484f58;white-space:nowrap">{{add $i 1}}</td>
      <td class="content-cell">{{$item.Content}}</td>
      <td><span class="risk-badge {{riskClass $item.Confidence}}">{{riskLabel $item.Confidence}}</span></td>
      <td style="text-align:center;color:#8b949e">{{$item.Count}}</td>
      <td class="path-cell">{{$item.FilePath}}</td>
      <td style="text-align:center;color:#8b949e">{{$item.LineNumber}}</td>
      <td class="ctx-cell">{{$item.Context}}</td>
    </tr>
    {{end}}
    </tbody>
  </table>
  </div>
  {{end}}
</div>
{{end}}

</div>
<div class="footer">Generated by <b>Gwxapkg</b> · <a href="https://github.com/25smoking/Gwxapkg" style="color:#58a6ff;text-decoration:none">github.com/25smoking/Gwxapkg</a></div>

<script>
var currentTab='all';
function switchTab(key,el){
  document.querySelectorAll('.panel').forEach(p=>p.classList.remove('active'));
  document.querySelectorAll('.tab').forEach(t=>t.classList.remove('active'));
  document.getElementById('panel-'+key).classList.add('active');
  el.classList.add('active');
  currentTab=key;
  filterTable();
}
function filterTable(){
  var q=document.getElementById('search').value.toLowerCase();
  var tbl=document.getElementById('tbl-'+currentTab);
  if(!tbl)return;
  tbl.querySelectorAll('tbody tr').forEach(function(row){
    row.style.display=row.innerText.toLowerCase().includes(q)?'':'none';
  });
}
</script>
</body>
</html>`
