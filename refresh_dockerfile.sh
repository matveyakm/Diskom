#!/bin/bash
# test 2
# Имя нового сервиса/модуля
NEW_NAME=$1
# Тип нового сервиса/модуля
NEW_TYPE=$2

 echo "--- Config: SubDir=$NEW_TYPE | Name=$NEW_NAME ---"
if [ -z "$NEW_NAME" ]; then
    echo "Ошибка: укажите имя нового сервиса"
    exit 1
fi

# Копируем шаблон и заменяем старое имя на новое
# g - заменить все вхождения в файле
#sed "s/{msgo}/$NEW_NAME/g; s/{type}/$NEW_TYPE/g" ./Dockerfiles/Dockerfile.template > ./Dockerfiles/$NEW_NAME
sed "s/{msgo}/$NEW_NAME/g; s/{type}/$NEW_TYPE/g" ./Dockerfiles/Dockerfile.work > ./Dockerfiles/$NEW_NAME
