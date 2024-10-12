package main

import (
	"io"
	"os"
	"fmt"
	"reflect"
	"strconv"
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

func loadFile(fileName string) ([]byte, error) {
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Could not open assets/all-words.txt")
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Could not read word list from assets/all-words.txt")
		return nil, err
	}

    return data, nil
}

func makeDefaultConfig() Config {
    return Config{
        "assets/all-words.txt",
        "assets/Cantarell_700Bold.ttf",
        "assets/Cabin-SemiBold.ttf",
        [4]string{"real", "real", "none", "none"},
        "classic",
        120,
    }
}

func saveConfig(config *Config) {
    configFields := reflect.ValueOf(config).Elem()
    configType := configFields.Type()
    nConfigKeys := configFields.NumField()
    var builder strings.Builder

    for i := 0; i < nConfigKeys; i++ {
        field := configFields.Field(i)
        t := field.Type().Name()
        builder.WriteString(configType.Field(i).Name)
        if t == "int" {
            builder.WriteString(strconv.Itoa(field.Interface().(int)))
        } else if t == "string" {
            builder.WriteString(field.Interface().(string))
        } else {
            builder.WriteString(strings.Join(field.Interface().([]string), " "))
        }
    }

    os.WriteFile("config.txt", []byte(builder.String()), 0666)
}

func loadConfig() (config Config, assets Assets, err error) {
    configFields := reflect.ValueOf(&config).Elem()
    assetsFields := reflect.ValueOf(&assets).Elem()

    configType := configFields.Type()
    configKeys := make(map[string]int)
    nConfigKeys := configFields.NumField()
    for i := 0; i < nConfigKeys; i++ {
        configKeys[configType.Field(i).Name] = i
    }

    assetType := assetsFields.Type()
    assetKeys := make(map[string]int)
    nAssetKeys := assetsFields.NumField()
    for i := 0; i < nAssetKeys; i++ {
        assetKeys[assetType.Field(i).Name] = i
    }

    configData, err := loadFile("config.txt")
    if configData != nil {
        lines := strings.Split(string(configData), "\n")
        for _, l := range lines {
            idx := strings.IndexByte(l, ' ')
            if idx <= 0 {
                continue
            }
            name := l[:idx]
            configIdx := configKeys[name]

            if strings.Contains(name, "Arr") {
                values := strings.Split(l[idx+1:], " ")
                if len(values) != 4 {
                    return Config{}, Assets{}, error("field \"" + name + "\" contains " + len(values) + " fields, not 4\n")
                }

                arr := configFields.Field(configIdx).Interface()
                for i := 0; i < 4; i++ {
                    arr[i] = values[i]
                }
            } else if strings.Contains(name, "Int") {
                n, err := strconv.Atoi(l[idx+1:])
                if err != nil {
                    return Config{}, Assets{}, err
                }

                configFields.Field(configIdx).SetInt(n)
            } else {
                configFields.Field(configIdx).SetString(l[idx+1:])
            }
        }
    } else {
        config = makeDefaultConfig()
        saveConfig(&config)
    }

    for i := 0; i < nAssetKeys; i++ {
        assetName := assetType.Field(i).Name
        name := assetName + "File"
        configIdx := configKeys[name]
        fileName := configFields.Field(configIdx).Interface()
        data, err := loadFile(fileName)
        if data == nil {
            err = error("Failed to open " + name + " \"" + fileName + "\"")
            return Config{}, Assets{}, err
        }

        field := assetsFields.Field(i)
        if field.Type() != "[]byte" {
            field.Set(strings.Split(string(data), "\n"))
        } else {
            field.SetBytes(data)
        }
    }

    return config, assets, nil
}
