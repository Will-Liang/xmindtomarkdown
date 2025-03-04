package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Sheet 表示 content.json 数组中的每个思维导图页
type Sheet struct {
	ID        string `json:"id"`
	Class     string `json:"class"`
	RootTopic Topic  `json:"rootTopic"`
}

// Topic 表示每个节点
type Topic struct {
	ID             string `json:"id"`
	Class          string `json:"class"`
	Title          string `json:"title"`
	StructureClass string `json:"structureClass"`
	Branch         string `json:"branch,omitempty"`
	// 子节点 attached
	Children *Children `json:"children,omitempty"`
	// 分离的节点 detached
	Detached []Topic `json:"detached,omitempty"`
	// 节点链接，若存在则输出为超链接形式
	Href string `json:"href,omitempty"`
}

// Children 用于解析 children.attached 数组
type Children struct {
	Attached []Topic `json:"attached,omitempty"`
}

func main() {
	// 使用 flag 定义 -f 参数，但如果没有提供，则交互式提示用户输入
	var filePath string
	flag.StringVar(&filePath, "f", "", "指定要转换的 .xmind 文件路径")
	flag.Parse()

	if filePath == "" {
		fmt.Print("请输入 .xmind 文件路径: ")
		// 读取用户输入（去除两端空白字符）
		_, err := fmt.Scanln(&filePath)
		if err != nil || strings.TrimSpace(filePath) == "" {
			fmt.Println("必须指定 .xmind 文件路径")
			time.Sleep(600 * time.Second)
			os.Exit(1)
		}
	}

	// 打开 xmind 文件（ZIP 包）
	r, err := zip.OpenReader(filePath)
	if err != nil {
		fmt.Printf("打开文件失败: %v\n", err)
		time.Sleep(600 * time.Second)
		os.Exit(1)
	}
	defer r.Close()

	var contentJSON io.ReadCloser
	// 遍历压缩包，查找 content.json 文件
	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "content.json") {
			contentJSON, err = f.Open()
			if err != nil {
				fmt.Printf("打开 content.json 失败: %v\n", err)
				time.Sleep(600 * time.Second)
				os.Exit(1)
			}
			break
		}
	}
	if contentJSON == nil {
		fmt.Println("在 xmind 文件中未找到 content.json")
		time.Sleep(600 * time.Second)
		os.Exit(1)
	}
	defer contentJSON.Close()

	// 读取 content.json 内容
	data, err := io.ReadAll(contentJSON)
	if err != nil {
		fmt.Printf("读取 content.json 失败: %v\n", err)
		time.Sleep(600 * time.Second)
		os.Exit(1)
	}

	// 解析 JSON 数据（最外层为数组）
	var sheets []Sheet
	err = json.Unmarshal(data, &sheets)
	if err != nil {
		fmt.Printf("解析 JSON 失败: %v\n", err)
		time.Sleep(600 * time.Second)
		os.Exit(1)
	}

	// 生成 Markdown 输出文件，文件名与输入文件同名，仅扩展名变为 .md
	outFile := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".md"
	mdFile, err := os.Create(outFile)
	if err != nil {
		fmt.Printf("创建 Markdown 文件失败: %v\n", err)
		time.Sleep(600 * time.Second)
		os.Exit(1)
	}
	defer mdFile.Close()

	// 针对每个 sheet 输出 Markdown 内容
	for _, sheet := range sheets {
		// 根节点使用 h1 显示
		fmt.Fprintf(mdFile, "# %s\n\n", sheet.RootTopic.Title)

		// 输出 children.attached 节点，从递归层级0开始（对应标题 h2 开始）
		if sheet.RootTopic.Children != nil {
			for _, child := range sheet.RootTopic.Children.Attached {
				writeTopicMarkdown(mdFile, child, 0)
			}
		}
		// 输出 detached 节点（如果有），同样从层级0开始
		if len(sheet.RootTopic.Detached) > 0 {
			for _, child := range sheet.RootTopic.Detached {
				writeTopicMarkdown(mdFile, child, 0)
			}
		}
		// 分隔每个 sheet
		fmt.Fprintln(mdFile, "\n")
	}

	fmt.Printf("Markdown 文件已生成: %s\n", outFile)
}

// writeTopicMarkdown 根据节点类型和层级递归输出 Markdown 格式
func writeTopicMarkdown(w io.Writer, topic Topic, indent int) {
	if topic.Href != "" {
		// 超链接节点：依然普通文本输出
		//indentStr := strings.Repeat("  ", indent)
		//fmt.Fprintf(w, "%s- [%s](%s)\n", indentStr, topic.Title, topic.Href)
		topic.Title = strings.ReplaceAll(topic.Title, "\n", "")
		fmt.Fprintf(w, "[%s](%s)\n", topic.Title, topic.Href)
	} else {
		// 非超链接节点：使用标题输出，层级为 indent+2，最大为 h6
		headerLevel := indent + 2
		if headerLevel > 6 {
			headerLevel = 6
		}
		headerPrefix := strings.Repeat("#", headerLevel)
		fmt.Fprintf(w, "%s %s\n\n", headerPrefix, topic.Title)
	}

	// 递归输出 attached 子节点（层级加1）
	if topic.Children != nil {
		for _, child := range topic.Children.Attached {
			writeTopicMarkdown(w, child, indent+1)
		}
	}
	// 递归输出 detached 节点（层级加1）
	if len(topic.Detached) > 0 {
		for _, child := range topic.Detached {
			writeTopicMarkdown(w, child, indent+1)
		}
	}
}
