import random


def generate_random_string(length=5):
    characters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
    random_string = ''.join(random.choice(characters) for _ in range(length))
    return random_string


def generate_unique_link(base_url, length=8):
    # Generate a random string
    random_string = generate_random_string(length)
    return f"{base_url}/{random_string}"


def main():
    # Example usage
    base_url = "https://goto.conrad-klaus.de"
    for _ in range(10):
        print(generate_unique_link(base_url))


if __name__ == "__main__":
    main()
