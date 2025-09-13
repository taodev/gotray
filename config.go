package main

type CmdItem struct {
	Dir string   `yaml:"dir"`
	Cmd []string `yaml:"cmd"`
}

type MenuItem struct {
	Title string     `yaml:"name"`
	Icon  string     `yaml:"icon"`
	Cmd   []string   `yaml:"cmd"`
	Items []MenuItem `yaml:"items"`
}

type Config struct {
	Cmds map[string]CmdItem `yaml:"cmds"`
	Menu []MenuItem         `yaml:"menu"`
}
