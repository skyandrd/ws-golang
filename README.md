# About
Реализовать концепт сервера WS. 
Задача - получение через http POST запрос вида:

```json
{
    "device_id":"00000000-0000-1111-2222-334455667788","id":"ffffffff-0000-1111-2222-334455667788",
    "kind":1,
    "message":"Text message"
}
```

И отправка его на указанный device_id (переданный при регистрации устройства). В случае успешной доставки – ответ 200, в случае ошибки (можно, например, 404)
Если device_id не задан – то рассылка по всем зарегистрированным устройствам.

## Command example
```bash
curl --location 'http://localhost:8080/command' \
--header 'Content-Type: application/json' \
--data '{
    "device_id": "4f42afd3-41be-4f9e-9bbe-c02561e64fdf",
    "id": "ffffffff-0000-1111-2222-334455667788",
    "kind": 1,
    "message": "Text message"
}'
```
