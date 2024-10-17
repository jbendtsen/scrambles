package main

import (
	"io"
	"os"
	"fmt"
	"errors"
	"reflect"
	"strconv"
	"strings"
)

type Config struct {
    WordListFile string
    TilesFontFile string
    UiFontFile string
    PlayerTypesArr [4]string
    GameMode string
    TimeLimitSecondsInt int
}

type Assets struct {
    WordList []string
    TilesFont []byte
    UiFont []byte
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
        builder.WriteString(" ")
        if t == "int" {
            builder.WriteString(strconv.Itoa(field.Interface().(int)))
        } else if t == "string" {
            builder.WriteString(field.Interface().(string))
        } else {
            value := reflect.ValueOf(field.Interface())
            for j := 0; j < 4; j++ {
                if j != 0 {
                    builder.WriteString(" ")
                }
                builder.WriteString(value.Index(j).Interface().(string))
            }
        }
        builder.WriteString("\n")
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

    assetsType := assetsFields.Type()
    assetsKeys := make(map[string]int)
    nAssetsKeys := assetsFields.NumField()
    for i := 0; i < nAssetsKeys; i++ {
        assetsKeys[assetsType.Field(i).Name] = i
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
                    errMsg := "field \"" + name + "\" contains " + strconv.Itoa(len(values)) + " fields, not 4\n"
                    return Config{}, Assets{}, errors.New(errMsg)
                }

                var arr [4]string
                for i := 0; i < 4; i++ {
                    arr[i] = values[i]
                }
                configFields.Field(configIdx).Set(reflect.ValueOf(arr))
            } else if strings.Contains(name, "Int") {
                n, err := strconv.Atoi(l[idx+1:])
                if err != nil {
                    return Config{}, Assets{}, err
                }

                configFields.Field(configIdx).SetInt(int64(n))
            } else {
                configFields.Field(configIdx).SetString(l[idx+1:])
            }
        }
    } else {
        config = makeDefaultConfig()
        saveConfig(&config)
    }

    for i := 0; i < nAssetsKeys; i++ {
        assetName := assetsType.Field(i).Name
        name := assetName + "File"
        configIdx := configKeys[name]
        fileName := configFields.Field(configIdx).Interface().(string)
        data, err := loadFile(fileName)
        if data == nil {
            err = errors.New("Failed to open " + name + " \"" + fileName + "\"")
            return Config{}, Assets{}, err
        }

        t := assetsFields.Field(i).Type().String()
        fmt.Println(t)
        if t != "[]byte" && t != "[]uint8" {
            arr := strings.Split(string(data), "\n")
            assetsFields.Field(i).Set(reflect.ValueOf(arr))
        } else {
            assetsFields.Field(i).SetBytes(data)
        }
    }

    return config, assets, nil
}
