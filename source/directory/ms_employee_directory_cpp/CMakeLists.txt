cmake_minimum_required(VERSION 4.2.3)

set(CMAKE_CXX_STANDARD 23)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_MODULE_STD OFF)
set(CMAKE_CXX_SCAN_FOR_MODULES ON)
#Sset(CMAKE_EXPERIMENTAL_CXX_IMPORT_STD "451f2fe2-a8a2-47c3-bc32-94786d8fc91b")
set(CMAKE_CXX_MODULE_STD OFF)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)
set(CMAKE_CXX_EXTENSIONS OFF)


project(Service CXX)


find_package(SQLite3 REQUIRED)
find_package(Crow REQUIRED)
find_package(simdjson REQUIRED)

add_executable(ms_employee_dir source/main.cpp)

target_link_libraries(ms_employee_dir PRIVATE SQLite::SQLite3 Crow::Crow simdjson::simdjson)

target_compile_options(ms_employee_dir PRIVATE
    -Wall
    -Wextra
    -Wpedantic
    -Werror
    -Wdangling
    -Wunsafe-buffer-usage
    -Warray-bounds
)


target_compile_options(ms_employee_dir PRIVATE
  $<$<CONFIG:Debug>:-fsanitize=address,undefined>
  $<$<CONFIG:Release>:-O3 -march=native>
)

target_link_options(ms_employee_dir PRIVATE
  $<$<CONFIG:Debug>:-fsanitize=address,undefined>
)
