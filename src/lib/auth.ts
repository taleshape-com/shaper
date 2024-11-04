export async function login(token: string) {
  return fetch(`/api/login/cookie`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ token }),
  }).then(async (response) => {
    if (response.status !== 200) {
      return response.json().then((data: { error: string }) => {
        return data.error;
      });
    }
    return;
  });
}

export async function testCookie() {
  return fetch(`/api/login/cookie/test`, {
    credentials: "include",
  }).then(async (response) => {
    if (response.status === 200) {
      return true;
    }
    if (response.status === 401) {
      return false;
    }
    return response
      .json()
      .then((data: { Error: { Type: number; Msg: string } }) => {
        throw new Error(data.Error.Msg);
      });
  });
}
