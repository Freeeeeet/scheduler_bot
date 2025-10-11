#!/usr/bin/env python3
"""
Скрипт для URL-кодирования пароля для PostgreSQL DSN
"""
import urllib.parse
import sys


def encode_password(password):
    """URL-кодирует пароль для использования в DSN строке"""
    return urllib.parse.quote(password, safe="")


def main():
    print("🔐 URL Encoder для PostgreSQL DSN")
    print("=" * 50)

    if len(sys.argv) > 1:
        # Если пароль передан как аргумент
        password = sys.argv[1]
    else:
        # Интерактивный ввод
        password = input("Введите пароль: ")

    if not password:
        print("❌ Пароль не может быть пустым!")
        sys.exit(1)

    encoded = encode_password(password)

    print(f"\n✅ Результат:")
    print(f"Оригинал:  {password}")
    print(f"Закодирован: {encoded}")

    # Показываем примеры замен
    print(f"\n📝 Что изменилось:")
    changes = []
    for orig_char, enc_char in zip(password, encoded):
        if orig_char != enc_char:
            # Находим закодированную версию этого символа
            encoded_char = urllib.parse.quote(orig_char, safe="")
            if encoded_char != orig_char:
                changes.append(f"  {orig_char}  →  {encoded_char}")

    if changes:
        for change in set(changes):
            print(change)
    else:
        print("  (специальные символы не найдены)")

    print(f"\n🔗 Используйте в DSN:")
    print(f"DB_DSN=postgres://username:{encoded}@host:5432/database")


if __name__ == "__main__":
    main()
