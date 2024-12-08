#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <pthread.h>

#define PORT 9911
#define BUFFER_SIZE 4096

void *handle_client(void *client_socket_ptr) {
    int client_socket = *(int*)client_socket_ptr;
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

    close(client_socket);
    return NULL;
}

int main() {
    int server_socket, client_socket;
    struct sockaddr_in server_addr, client_addr;
    socklen_t client_addr_len = sizeof(client_addr);

    if ((server_socket = socket(AF_INET, SOCK_STREAM, 0)) < 0) {
        perror("Socket creation failed");
        return 1;
    }

    server_addr.sin_family = AF_INET;
    server_addr.sin_addr.s_addr = INADDR_ANY;
    server_addr.sin_port = htons(PORT);

    if (bind(server_socket, (struct sockaddr*)&server_addr, sizeof(server_addr)) < 0) {
        perror("Bind failed");
        close(server_socket);
        return 1;
    }

    if (listen(server_socket, 10) < 0) {
        perror("Listen failed");
        close(server_socket);
        return 1;
    }

    printf("Server: http://localhost:%d\n", PORT);

    while (1) {

        client_socket = accept(server_socket, (struct sockaddr*)&client_addr, &client_addr_len);
        if (client_socket < 0) {
            perror("Accept failed");
            continue;
        }

        int *client_socket_ptr = malloc(sizeof(int));
        *client_socket_ptr = client_socket;

        pthread_t thread_id;
        if (pthread_create(&thread_id, NULL, handle_client, client_socket_ptr) != 0) {
            perror("Thread creation failed");
            free(client_socket_ptr);
            close(client_socket);
        } else {
            pthread_detach(thread_id);
        }
    }

    close(server_socket);
    return 0;
}
