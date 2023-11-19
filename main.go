package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	input "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const listHeight = 14
const defaultWidth = 20

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			PaddingTop(1).
			PaddingBottom(1).
			PaddingLeft(2).
			Width(26)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type item string

func (i item) FilterValue() string { return "" }

type Favorite struct {
	Title string `json:"title"`
	Url   string `json:"url"`
}

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type model struct {
	list       list.Model
	favorites  []Favorite
	url        string
	choice     string
	quitting   bool
	mode       string
	titleInput input.Model
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// タイトル追加モード
	if m.mode == "addTitle" {
		return m.UpdateAddItem(msg)
	}

	// 一覧モード
	if m.mode == "list" {
		return m.UpdateList(msg)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// 一覧モードUpdate
func (m model) UpdateList(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		// 起動時にウィンドウサイズを設定
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		// キャンセル
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		// エンター
		case "enter":
			item, ok := m.list.SelectedItem().(item)
			for _, f := range m.favorites {
				if f.Title == string(item) {
					m.url = f.Url
				}
			}
			if ok {
				m.choice = string(item)
				exec.Command("open", m.url).Start()
			}
			return m, tea.Quit
		// 追加モード
		case "i":
			m.mode = "addTitle"
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	return m, cmd
}

// 追加モードUpdate
func (m model) UpdateAddItem(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+q":
			m.mode = "list"
			m.titleInput.Reset()
			return m, nil
		case "enter":

			// 空文字ならリストに戻る
			if m.titleInput.Value() == "" {
				m.mode = "list"
				return m, nil
			}

			// リストを更新
			m.favorites = append(m.favorites, Favorite{
				Title: m.titleInput.Value(),
				Url:   "URL",
			})
			// リストをJSONにエンコード
			jsonData, err := json.MarshalIndent(m.favorites, "", "    ")
			if err != nil {
				log.Fatal(err)
			}
			// ファイルに書き込む
			err = ioutil.WriteFile("favorites.json", jsonData, os.ModePerm)
			if err != nil {
				log.Fatal(err)
			}

			// list.Add的な項目追加の関数はないためNewで再生成
			var items []list.Item
			for _, f := range m.favorites {
				items = append(items, item(f.Title))
			}
			m.list = list.New(items, itemDelegate{}, defaultWidth, listHeight)
			m.mode = "list"
			m.titleInput.Reset()
			return m, nil
		}

	}
	var cmd tea.Cmd
	m.titleInput, cmd = m.titleInput.Update(msg)
	return m, cmd
}

/*
一覧モードView
*/
func (m model) View() string {
	if m.mode == "addTitle" {
		return m.addingTaskView()
	}
	if m.choice != "" {
		return quitTextStyle.Render(fmt.Sprintf("%s(%s) を選択", m.choice, m.url))
	}
	if m.quitting {
		return quitTextStyle.Render(fmt.Sprintf("キャンセルしました! %v", m.favorites))
	}
	return "\n" + m.list.View()
}

// 追加モードView
func (m model) addingTaskView() string {
	return fmt.Sprintf("Additional Mode\n\nInput a new task name\n\n " + m.titleInput.View() + "\n\nPress Ctrl+Q to back to normal mode")
}

func main() {

	// お気に入りリストの読み込み
	raw, err := ioutil.ReadFile("favorites.json")
	if err != nil {
		panic(err)
	}
	var favorites []Favorite
	json.Unmarshal(raw, &favorites)

	var items []list.Item
	for _, f := range favorites {
		items = append(items, item(f.Title))
	}

	// 一覧モデルの設定
	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "🌷 My Favorite Links"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	// テキストインプットモデルの設定
	ti := textinput.New()
	ti.Placeholder = "Write New Task Name"
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = 50

	m := model{
		list:       l,
		favorites:  favorites,
		titleInput: ti,
	}
	m.mode = "list"

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
