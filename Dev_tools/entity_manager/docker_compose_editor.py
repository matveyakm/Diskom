import re
import os
from pathlib import Path

PROJECT_ROOT = Path(__file__).parent.parent.parent

COMPOSE_PATH = PROJECT_ROOT / "docker-compose.yaml"
GO_WORK_PATH = PROJECT_ROOT / "source" / "go.work"
CORS_ORIGINS = "https://discom.spbgu.localhost:8443,https://old.discom.spbgu.localhost:8443"


def get_compose_content():
    if not COMPOSE_PATH.exists():
        return ""
    return COMPOSE_PATH.read_text()


def write_compose_content(content):
    COMPOSE_PATH.write_text(content)


def get_next_ips():
    """Find the highest IP in docker-compose and return next (postgres_ip, api_ip)."""
    content = get_compose_content()
    ip_pattern = re.compile(r"ipv4_address:\s*172\.25\.0\.(\d+)")
    matches = ip_pattern.findall(content)
    ips = [int(m) for m in matches]
    if not ips:
        return "172.25.0.62", "172.25.0.63"

    max_ip = max(ips)

    # system_object_directory uses 99 and 100, so next should be 101/102
    # but there's also space between 61 and 99
    # Let's use the next gap after max_ip
    used = set(ips)
    next_ip = max_ip + 1
    while next_ip in used:
        next_ip += 1

    postgres_ip = next_ip
    api_ip = next_ip + 1
    while api_ip in used or api_ip == postgres_ip:
        api_ip += 1

    return f"172.25.0.{postgres_ip}", f"172.25.0.{api_ip}"


def add_postgres_service_block(ref, db_ip, category):
    ms_name = f"ms_{ref}"
    source_path = f"./source/{category}/{ms_name}"
    return f'''  postgres_{ms_name}:
    image: postgres:latest
    container_name: postgres_{ref}
    command: postgres -c listen_addresses={db_ip} -c port=5434
    restart: always
    env_file:
      - {source_path}/parameters.env
    volumes:
      - postgres_{ms_name}:/var/lib/postgresql
      - {source_path}/dbshell:/docker-entrypoint-initdb.d/
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5
    networks:
      vnet:
        ipv4_address: {db_ip}

'''


def get_traefik_labels(ref, api_ip):
    return f'''      - "traefik.enable=true"
      - "traefik.http.routers.api_ms_{ref}.rule=Host(`discom.spbgu.localhost`) && PathPrefix(`/api/{ref}`)"
      - "traefik.http.services.api_ms_{ref}.loadbalancer.server.port=8443"
      - "traefik.http.routers.api_ms_{ref}.entrypoints=websecure"
      - "traefik.http.routers.api_ms_{ref}.tls=true"
      - "traefik.http.middlewares.backend-cors.headers.accesscontrolalloworiginlist={CORS_ORIGINS}"
      - "traefik.http.middlewares.backend-cors.headers.accesscontrolallowmethods=GET,POST,OPTIONS,PUT,DELETE"
      - "traefik.http.middlewares.backend-cors.headers.accesscontrolallowheaders=DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization"
      - "traefik.http.middlewares.backend-cors.headers.accesscontrolmaxage=1728000"
      - "traefik.http.routers.api_ms_{ref}.middlewares=backend-cors"
'''


def add_api_service_block(ref, api_ip, category):
    ms_name = f"ms_{ref}"
    source_path = f"./source/{category}/{ms_name}"
    return f'''  api_{ms_name}:
    build:
      context: .
      dockerfile: ./Dockerfiles/{ms_name}
    container_name: api_{ms_name}
    restart: always
    env_file:
      - {source_path}/parameters.env
    networks:
      vnet:
        ipv4_address: {api_ip}
    labels:
{get_traefik_labels(ref, api_ip)}
'''


def add_volume_entries(ref):
    ms_name = f"ms_{ref}"
    return f'''  postgres_{ms_name}:
  api_{ms_name}:
'''


def add_microservice_to_compose(ref, category):
    """Add postgres and api service blocks to docker-compose.yaml."""
    content = get_compose_content()
    if not content:
        return False, "docker-compose.yaml not found"

    # Check if already exists
    if f"ms_{ref}" in content:
        return False, f"Service ms_{ref} already exists in docker-compose.yaml"

    db_ip, api_ip = get_next_ips()

    # Find insertion point: after the last microservice definition, before volumes
    # Look for the volumes section
    volumes_match = re.search(r"\nvolumes:\n", content)
    if not volumes_match:
        return False, "Could not find volumes section in docker-compose.yaml"

    insert_pos = volumes_match.start() + 1

    postgres_block = add_postgres_service_block(ref, db_ip, category)
    api_block = add_api_service_block(ref, api_ip, category)
    combined_block = postgres_block + "\n" + api_block

    new_content = content[:insert_pos] + "\n" + combined_block + content[insert_pos:]
    write_compose_content(new_content)

    # Now add volumes
    add_volumes(ref)

    return True, f"Services added: postgres at {db_ip}, api at {api_ip}"


def add_volumes(ref):
    """Add volume entries to the volumes section."""
    content = get_compose_content()
    if not content:
        return

    ms_name = f"ms_{ref}"
    vol_entries = add_volume_entries(ref)

    # Add before the network section
    net_match = re.search(r"\n# --- Shared resources ---", content)
    if net_match:
        new_content = content[:net_match.start()] + "\n" + vol_entries + content[net_match.start():]
        write_compose_content(new_content)


def add_to_go_work(ref, category):
    """Add microservice to source/go.work use block."""
    content = GO_WORK_PATH.read_text()

    entry = f"./{category}/ms_{ref}"
    if entry in content:
        return False, f"{entry} already in go.work"

    # Find the use block and add the entry
    use_match = re.search(r"use\s*\(([^)]*)\)", content, re.DOTALL)
    if not use_match:
        return False, "Could not find use block in go.work"

    existing = use_match.group(1)
    new_existing = existing.rstrip() + f"\n\t./{category}/ms_{ref}\n"

    new_content = content[:use_match.start(1)] + new_existing + content[use_match.end(1):]
    GO_WORK_PATH.write_text(new_content)

    return True, f"Added {entry} to go.work"
