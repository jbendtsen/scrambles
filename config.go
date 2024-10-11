package main

import (
	"io"
	"os"
	"reflect"
	"strings"
)

type Config struct {
    wordListFile string
    tilesFontFile string
    uiFontFile string
    playerTypesArr [4]string
    gameMode string
    timeLimitSecondsInt int
}

type Assets struct {
    wordList []string
    tilesFont []byte
    uiFont []byte
}

func loadFile(fileName string) []byte, error {
	wordsFile, err := os.Open()
	if err != nil {
		fmt.Println("Could not open assets/all-words.txt")
		return nil
	}
	defer wordsFile.Close()

	wordsBytes, err := io.ReadAll(wordsFile)
	if err != nil {
		fmt.Println("Could not read word list from assets/all-words.txt")
		return nil
	}
}

func loadTtfData() []byte {
	entries, err := os.ReadDir("assets/")
	if err != nil {
		fmt.Println("Could not open assets folder")
		return nil
	}

	var ttfName string
	for i := 0; i < len(entries); i++ {
		fname := entries[i].Name()
		if strings.HasSuffix(fname, ".ttf") {
			ttfName = fname
			break
		}
	}

	if ttfName == "" {
		fmt.Println("Could not a ttf file in the assets folder")
		return nil
	}

	f, err := os.Open("assets/" + ttfName)
	if err != nil {
		fmt.Println("Could not open assets/" + ttfName)
		return nil
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		fmt.Println("Could not read font from assets/" + ttfName)
		return nil
	}

	return data
}

func loadWords() byte[] {
    data, err := loadFile("assets/all-words.txt")
	return strings.Split(string(wordsBytes), "\n")
}

func makeDefaultConfig() Config {
    return {
        "assets/all-words.txt",
        "assets/Cantarell_700Bold.ttf",
        "assets/Cabin-SemiBold.ttf",
        "real real none none",
        "classic",
        120
    }
}

func loadConfig() (config Config, assets Assets, err error) {
    configFields = reflect.ValueOf(&config).Elem()
    assetsFields = reflect.ValueOf(&assets).Elem()

    configKeys := make(map[string]int)
    nConfigKeys := configFields.NumField()
    for i := 0; i < nConfigKeys; i++ {
        configKeys[configFields.Field(i).Name] = i
    }

    assetKeys := make(map[string]int)
    nAssetKeys := assetFields.NumField()
    for i := 0; i < nAssetKeys; i++ {
        assetKeys[assetFields.Field(i).Name] = i
    }

    configData, err := loadFile("assets/config.txt")
    if configData != nil {
        lines := strings.Split(string(wordsBytes), "\n")
        for l := range lines {
            idx := strings.IndexByte(l, ' ')
            if idx <= 0 {
                continue
            }
            name := l[:idx]
            configIdx := configKeys[name]
            if name.Contains("Arr") {
                values := strings.Split(l[idx+1:], " ")
                if len(values) == 4 {
                    arr := configFields.Field(configIdx).Interface()
                    for i := 0; i < 4; i++ {
                        arr[i] = values[i]
                    }
                }
            } else if name.Contains("Int") {
                n, err := strconv.Atoi(l[idx+1:])
                if err == nil {
                    configFields.Field(configIdx).SetInt(n)
                }
            } else {
                configFields.Field(configIdx).SetString(l[idx+1:])
            }
        }
    } else {
        config = makeDefaultConfig()
    }

    for i := 0; i < nAssetKeys; i++ {
        name := l[:idx] + "File"
        configIdx := configKeys[name]
        fileName := configFields.Field(configIdx).Interface()
        data, err := loadFile(fileName)
        if data != nil {
            field := assetFields.Field(i)
            if field.Type() != "[]byte" {
                field.Set(strings.Split(string(data), "\n"))
            } else {
                field.SetBytes(data)
            }
        }
    }

    return config, assets, nil
}
