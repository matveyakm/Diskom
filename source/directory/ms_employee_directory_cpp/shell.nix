{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell.override { stdenv = pkgs.llvmPackages_18.stdenv; } {
  # Инструменты сборки (работают на хосте сборщика)
  nativeBuildInputs = with pkgs; [
    cmake                 # Генератор проектов
    ninja                 # Быстрый сборщик вместо make
    just                  # Раннер команд
    pkg-config            # Поиск системных библиотек
    conan                 # Менеджер пакетов C++
  ];

  # Зависимости времени выполнения (передаются компилятору).
  # Все основные C++ библиотеки (crow, simdjson, sqlite3) ставит Conan.
  # Оставляем здесь только системные базовые зависимости, необходимые для сборки Conan/Сmake.
  buildInputs = with pkgs; [
    openssl               # Нужен для сетевых запросов и шифрования (Crow/Conan)
    sqlite                # Нужен для утилиты командной строки в скрипте db.sh
  ];

  shellHook = ''
    # Фиксируем компиляторы Nix для процессов Conan
    export CC=clang
    export CXX=clang++

    # Подавляем ошибку неиспользованных аргументов (из-за -stdlib=libstdc++ в Conan)
    export NIX_CFLAGS_COMPILE="-Wno-error=unused-command-line-argument $NIX_CFLAGS_COMPILE"

    # Отключаем Precompiled Headers (PCH)
    export CONAN_CMAKE_GENERATOR="Ninja"
    export CONAN_CONF="tools.cmake.cmaketoolchain:user_toolchain_flags=['-DCMAKE_DISABLE_PRECOMPILE_HEADERS=ON']"

    # Очищаем переменные путей инклудов, чтобы обертка Nix сама управляла заголовками
    unset C_INCLUDE_PATH
    unset CXX_INCLUDE_PATH
    unset CPATH

    # Автоматически исправляем профиль Conan, меняя проблемный libc++ на стандартный libstdc++11
    if [ -f /root/.conan2/profiles/default ]; then
      sed -i 's/compiler.libcxx=libc++/compiler.libcxx=libstdc++11/g' /root/.conan2/profiles/default
      echo "=== Профиль Conan успешно переведен на libstdc++11 ==="
    fi

    echo "=== Nix-окружение (Clang 18 + Wno-error) успешно загружено ==="
  '';
}
