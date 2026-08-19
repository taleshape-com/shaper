// SPDX-License-Identifier: MPL-2.0

function getRandomInt (max: number): number {
  if (max <= 0) {
    return 0;
  }
  const range = 0x100000000; // 2^32
  const limit = range - (range % max);
  const randomBuffer = new Uint32Array(1);
  let randomValue: number;
  do {
    crypto.getRandomValues(randomBuffer);
    randomValue = randomBuffer[0];
  } while (randomValue >= limit);
  return randomValue % max;
}

export function generatePassword (length: number = 16): string {
  if (length <= 0) {
    return "";
  }

  const uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";
  const lowercase = "abcdefghijklmnopqrstuvwxyz";
  const digits = "0123456789";
  const allChars = uppercase + lowercase + digits;

  if (length < 3) {
    const chars: string[] = [];
    for (let i = 0; i < length; i++) {
      chars.push(allChars[getRandomInt(allChars.length)]);
    }
    return chars.join("");
  }

  // Ensure at least one character from each category
  const password = [
    uppercase[getRandomInt(uppercase.length)],
    lowercase[getRandomInt(lowercase.length)],
    digits[getRandomInt(digits.length)],
  ];

  // Fill the remaining length with random characters
  for (let i = 3; i < length; i++) {
    password.push(allChars[getRandomInt(allChars.length)]);
  }

  // Shuffle the array to avoid predictable patterns
  for (let i = password.length - 1; i > 0; i--) {
    const j = getRandomInt(i + 1);
    [password[i], password[j]] = [password[j], password[i]];
  }

  return password.join("");
}
