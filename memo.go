// switch msg := msg.(type) {
// case tea.KeyMsg:
// 	switch msg.String() {
// 	case "ctrl+q":
// 		m.mode = "list"
// 		m.inputs[0].Reset()
// 		return m, nil
// 	case "enter":

// 		// 空文字ならリストに戻る
// 		if m.inputs[0].Value() == "" {
// 			m.mode = "list"
// 			return m, nil
// 		}

// 		// リストを更新
// 		m.favorites = append(m.favorites, Favorite{
// 			Title: m.inputs[0].Value(),
// 			Url:   "URL",
// 		})
// 		// リストをJSONにエンコード
// 		jsonData, err := json.MarshalIndent(m.favorites, "", "    ")
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		// ファイルに書き込む
// 		err = ioutil.WriteFile("favorites.json", jsonData, os.ModePerm)
// 		if err != nil {
// 			log.Fatal(err)
// 		}

// 		// list.Add的な項目追加の関数はないためNewで再生成
// 		var items []list.Item
// 		for _, f := range m.favorites {
// 			items = append(items, item(f.Title))
// 		}
// 		m.list = list.New(items, itemDelegate{}, defaultWidth, listHeight)
// 		m.mode = "list"
// 		m.inputs[0].Reset()
// 		return m, nil
// 	}

// }
// var cmd tea.Cmd
// // m.inputs, cmd = m.inputs[0].Update(msg)
// return m, cmd