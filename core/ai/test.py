import requests
import base64
import json

class DeepSeekClient:
    def __init__(self, credential_string):
        self.proxy_url = "https://deproxy.kchugalinskiy.ru"
        self.credential_string = credential_string
        
        # Пробуем разные варианты парсинга credentials
        if "@" in credential_string:
            # Вариант 1: user@password
            user, password = credential_string.split("@", 1)
            self.auth_header = f"Basic {base64.b64encode(f'{user}:{password}'.encode()).decode()}"
        else:
            # Вариант 2: вся строка как пароль или токен
            self.auth_header = f"Basic {base64.b64encode(credential_string.encode()).decode()}"
        
        self.system_prompt = "Ты - полезный AI-ассистент. Отвечай точно и по делу."
    
    def _debug_response(self, response, endpoint_name):
        """Отладочная информация о ответе"""
        print(f"\n=== DEBUG {endpoint_name} ===")
        print(f"Status Code: {response.status_code}")
        print(f"Headers: {dict(response.headers)}")
        print(f"Content: {response.text}")
        print("=== END DEBUG ===\n")
    
    def check_status(self):
        """Проверяет статус токена"""
        headers = {"Authorization": self.auth_header}
        
        try:
            response = requests.get(
                f"{self.proxy_url}/deeproxy/api/status", 
                headers=headers,
                timeout=10
            )
            self._debug_response(response, "GET /status")
            
            if response.status_code == 200 and response.text.strip():
                return response.json()
            else:
                return {
                    "error": f"HTTP {response.status_code}", 
                    "content": response.text,
                    "auth_used": self.credential_string
                }
                
        except Exception as e:
            return {"error": str(e)}

# Тестируем все варианты credentials
if __name__ == "__main__":
    credentials_list = [
        "41-1@SjA9YW9S",
        "41-2@U0dMUjFs", 
        "42@dkljRktA"
    ]
    
    for creds in credentials_list:
        print(f"\n{'='*50}")
        print(f"Тестируем credentials: {creds}")
        print(f"{'='*50}")
        
        client = DeepSeekClient(creds)
        status = client.check_status()
        print(f"Результат: {status}")