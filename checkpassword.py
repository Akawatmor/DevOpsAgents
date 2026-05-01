

def checkPassword(password):
    if len(password) < 8:
        return False
    if not any(char.isdigit() for char in password):
        return False
    if not any(char.isalpha() for char in password):
        return False
    return True




password = input("Enter a password: ")
if checkPassword(password):
    print("Password is strong.")
else:
    print("Password is weak. It must be at least 8 characters long and include digits.")