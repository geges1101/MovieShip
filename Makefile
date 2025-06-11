.PHONY: run build tidy docker-up docker-down clean

# Запуск приложения локально
run:
	env $$(cat config.env | xargs) go run main.go

# Сборка бинарника
build:
	go build -o movieship main.go

# Установка/обновление зависимостей
tidy:
	go mod tidy

# Запуск через docker-compose
docker-up:
	docker-compose up --build

# Остановка docker-compose
docker-down:
	docker-compose down

# Очистка скомпилированных файлов
clean:
	rm -f movieship
