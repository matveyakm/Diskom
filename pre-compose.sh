#!/bin/bash
clear
sudo docker compose stop
sudo docker compose down --volumes --rmi local

# 1. Список модулей (путь и имя модуля через двоеточие)
# Можно легко добавлять новые или комментировать ненужные
MODULES=(
    "source/document/ms_document:ms_document"
    "source/document/ms_appointment_document:ms_appointment_document"
    "source/document/ms_dismissal_document:ms_dismissal_document"
    "source/document/ms_acceptance_document:ms_acceptance_document"
    "source/document/ms_create_department_document:ms_create_department_document"
    "source/document/ms_create_job_title_document:ms_create_job_title_document"
    "source/document/ms_create_counterparty_document:ms_create_counterparty_document"
    "source/document/ms_create_organization_document:ms_create_organization_document"
    "source/document/ms_create_regulations_clauses_document:ms_create_regulations_clauses_document"
    "source/directory/ms_employee_directory:ms_employee_directory"
    "source/directory/ms_department_directory:ms_department_directory"
    "source/directory/ms_regulations_clauses_directory:ms_regulations_clauses_directory"
    "source/directory/ms_organization_directory:ms_organization_directory"
    "source/directory/ms_counterparty_directory:ms_counterparty_directory"
    "source/directory/ms_job_title_directory:ms_job_title_directory"
    "source/directory/ms_disciplinary_measures:ms_disciplinary_measures"
    "source/directory/ms_reasons_for_the_dcs_disagreement:ms_reasons_for_the_dcs_disagreement"
    "source/directory/ms_system_object_directory:ms_system_object_directory"
    "source/ledger/ms_employee_data_ledger:ms_employee_data_ledger"
    "source/ledger/ms_staff_placement_ledger:ms_staff_placement_ledger"
    "source/ledger/ms_department_data_ledger:ms_department_data_ledger"
    "source/ledger/ms_job_title_data_ledger:ms_job_title_data_ledger"
    "source/ledger/ms_organization_data_ledger:ms_organization_data_ledger"
    "source/report/ms_job_report:ms_job_report"
    ""
)

cd ./source/shared/
rm -f go.mod go.sum
go mod init shared
go mod tidy
cd - > /dev/null # Возвращаемся обратно в корень

# 2. Цикл обработки
for ENTRY in "${MODULES[@]}"; do

    # Разделяем путь и название
    DIR="${ENTRY%%:*}"
    MOD_NAME="${ENTRY##*:}"

    PARENT_DIR=$(dirname "$DIR")
    MOD_SUBDIR=$(basename "$PARENT_DIR")
    bash ./refresh_dockerfile.sh "$MOD_NAME" "$MOD_SUBDIR"

    echo "--- Пересобираю модуль: $MOD_NAME ---"

    cd "./$DIR" || continue
    rm -f go.mod go.sum
    go mod init "$MOD_NAME"
    go mod tidy
    cd - > /dev/null # Возвращаемся обратно в корень
done

# 3. Сборка db.sh из system_description/ файлов
echo "--- Собираю db.sh для system_object_directory ---"
bash source/directory/ms_system_object_directory/build-db.sh

# 4. Запуск Docker
sudo docker compose up -d
