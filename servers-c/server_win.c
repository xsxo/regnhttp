#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <winsock2.h>
#include <windows.h>

#pragma comment(lib, "ws2_32.lib") 

#define PORT 80
#define BUFFER_SIZE 4096

DWORD WINAPI handle_client(LPVOID client_socket_ptr) {
    SOCKET client_socket = *(SOCKET*)client_socket_ptr;
    free(client_socket_ptr);

    char buffer[BUFFER_SIZE];

    while (1) {
        int bytes_read = recv(client_socket, buffer, BUFFER_SIZE - 1, 0);

        if (bytes_read <= 0) {
            break;
        }

        buffer[bytes_read] = '\0';

        char response[BUFFER_SIZE];
        snprintf(response, BUFFER_SIZE,
            "HTTP/1.1 200 OK\r\n"
            "Content-Length: %d\r\n"
            "Content-Type: text/plain\r\n"
            "\r\n"
            "%s",
            bytes_read, buffer);

        send(client_socket, response, strlen(response), 0);
    }

    closesocket(client_socket);
    return 0;
}

int main() {
    WSADATA wsa;
    SOCKET server_socket, client_socket;
    struct sockaddr_in server_addr, client_addr;
    int client_addr_len = sizeof(client_addr);

    if (WSAStartup(MAKEWORD(2, 2), &wsa) != 0) {
        printf("Failed. Error Code : %d\n", WSAGetLastError());
        return 1;
    }

    if ((server_socket = socket(AF_INET, SOCK_STREAM, 0)) == INVALID_SOCKET) {
        printf("Could not create socket: %d\n", WSAGetLastError());
        WSACleanup();
        return 1;
    }

    server_addr.sin_family = AF_INET;
    server_addr.sin_addr.s_addr = INADDR_ANY;
    server_addr.sin_port = htons(PORT);

    if (bind(server_socket, (struct sockaddr*)&server_addr, sizeof(server_addr)) == SOCKET_ERROR) {
        printf("Bind failed: %d\n", WSAGetLastError());
        closesocket(server_socket);
        WSACleanup();
        return 1;
    }

    listen(server_socket, 10);
    printf("Server: http://localhost:%d\n", PORT);

    while (1) {
        client_socket = accept(server_socket, (struct sockaddr*)&client_addr, &client_addr_len);
        if (client_socket == INVALID_SOCKET) {
            printf("Accept failed: %d\n", WSAGetLastError());
            continue;
        }

        SOCKET* client_socket_ptr = malloc(sizeof(SOCKET));
        *client_socket_ptr = client_socket;

        CreateThread(NULL, 0, handle_client, client_socket_ptr, 0, NULL);
    }

    closesocket(server_socket);
    WSACleanup();
    return 0;
}