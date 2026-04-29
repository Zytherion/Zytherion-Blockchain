use std::panic;
use std::slice;

use tfhe::prelude::*;
use tfhe::{generate_keys, set_server_key, ConfigBuilder, FheUint32};

fn write_bytes_out(bytes: Vec<u8>, out_ptr: *mut *mut u8, out_len: *mut usize) {
    let len = bytes.len();
    let mut boxed = bytes.into_boxed_slice();
    let ptr = boxed.as_mut_ptr();
    std::mem::forget(boxed);
    unsafe {
        *out_ptr = ptr;
        *out_len = len;
    }
}

#[no_mangle]
pub extern "C" fn tfhe_generate_keys(
    out_client_key: *mut *mut u8,
    out_client_key_len: *mut usize,
    out_server_key: *mut *mut u8,
    out_server_key_len: *mut usize,
) -> i32 {
    let result = panic::catch_unwind(|| {
        let config = ConfigBuilder::default().build();
        let (client_key, server_key) = generate_keys(config);
        let ck_bytes = bincode::serialize(&client_key).map_err(|_| ())?;
        let sk_bytes = bincode::serialize(&server_key).map_err(|_| ())?;
        Ok::<_, ()>((ck_bytes, sk_bytes))
    });
    match result {
        Ok(Ok((ck_bytes, sk_bytes))) => {
            write_bytes_out(ck_bytes, out_client_key, out_client_key_len);
            write_bytes_out(sk_bytes, out_server_key, out_server_key_len);
            0
        }
        _ => -1,
    }
}

#[no_mangle]
pub extern "C" fn tfhe_encrypt_u64(
    client_key_bytes: *const u8,
    client_key_len: usize,
    value: u64,
    out_ct: *mut *mut u8,
    out_ct_len: *mut usize,
) -> i32 {
    let result = panic::catch_unwind(|| {
        let ck_slice = unsafe { slice::from_raw_parts(client_key_bytes, client_key_len) };
        let client_key: tfhe::ClientKey = bincode::deserialize(ck_slice).map_err(|_| ())?;
        // Use FheUint32 for ~5KB ciphertexts; values are clamped to u32 range.
        let v32 = (value & 0xFFFF_FFFF) as u32;
        let ct = FheUint32::encrypt(v32, &client_key);
        let bytes = bincode::serialize(&ct).map_err(|_| ())?;
        Ok::<_, ()>(bytes)
    });
    match result {
        Ok(Ok(bytes)) => { write_bytes_out(bytes, out_ct, out_ct_len); 0 }
        _ => -1,
    }
}

#[no_mangle]
pub extern "C" fn tfhe_decrypt_u64(
    client_key_bytes: *const u8,
    client_key_len: usize,
    ct_bytes: *const u8,
    ct_len: usize,
    out_value: *mut u64,
) -> i32 {
    let result = panic::catch_unwind(|| {
        let ck_slice = unsafe { slice::from_raw_parts(client_key_bytes, client_key_len) };
        let ct_slice = unsafe { slice::from_raw_parts(ct_bytes, ct_len) };
        let client_key: tfhe::ClientKey = bincode::deserialize(ck_slice).map_err(|_| ())?;
        let ct: FheUint32 = bincode::deserialize(ct_slice).map_err(|_| ())?;
        let value: u32 = ct.decrypt(&client_key);
        Ok::<_, ()>(value as u64)
    });
    match result {
        Ok(Ok(value)) => { unsafe { *out_value = value }; 0 }
        _ => -1,
    }
}

#[no_mangle]
pub extern "C" fn tfhe_add(
    server_key_bytes: *const u8, server_key_len: usize,
    ct_a_bytes: *const u8, ct_a_len: usize,
    ct_b_bytes: *const u8, ct_b_len: usize,
    out_ct: *mut *mut u8, out_ct_len: *mut usize,
) -> i32 {
    let result = panic::catch_unwind(|| {
        let sk_slice = unsafe { slice::from_raw_parts(server_key_bytes, server_key_len) };
        let a_slice = unsafe { slice::from_raw_parts(ct_a_bytes, ct_a_len) };
        let b_slice = unsafe { slice::from_raw_parts(ct_b_bytes, ct_b_len) };
        let server_key: tfhe::ServerKey = bincode::deserialize(sk_slice).map_err(|_| ())?;
        set_server_key(server_key);
        let ct_a: FheUint32 = bincode::deserialize(a_slice).map_err(|_| ())?;
        let ct_b: FheUint32 = bincode::deserialize(b_slice).map_err(|_| ())?;
        let result_ct = ct_a + ct_b;
        let bytes = bincode::serialize(&result_ct).map_err(|_| ())?;
        Ok::<_, ()>(bytes)
    });
    match result {
        Ok(Ok(bytes)) => { write_bytes_out(bytes, out_ct, out_ct_len); 0 }
        _ => -1,
    }
}

#[no_mangle]
pub extern "C" fn tfhe_sub(
    server_key_bytes: *const u8, server_key_len: usize,
    ct_a_bytes: *const u8, ct_a_len: usize,
    ct_b_bytes: *const u8, ct_b_len: usize,
    out_ct: *mut *mut u8, out_ct_len: *mut usize,
) -> i32 {
    let result = panic::catch_unwind(|| {
        let sk_slice = unsafe { slice::from_raw_parts(server_key_bytes, server_key_len) };
        let a_slice = unsafe { slice::from_raw_parts(ct_a_bytes, ct_a_len) };
        let b_slice = unsafe { slice::from_raw_parts(ct_b_bytes, ct_b_len) };
        let server_key: tfhe::ServerKey = bincode::deserialize(sk_slice).map_err(|_| ())?;
        set_server_key(server_key);
        let ct_a: FheUint32 = bincode::deserialize(a_slice).map_err(|_| ())?;
        let ct_b: FheUint32 = bincode::deserialize(b_slice).map_err(|_| ())?;
        let result_ct = ct_a - ct_b;
        let bytes = bincode::serialize(&result_ct).map_err(|_| ())?;
        Ok::<_, ()>(bytes)
    });
    match result {
        Ok(Ok(bytes)) => { write_bytes_out(bytes, out_ct, out_ct_len); 0 }
        _ => -1,
    }
}

#[no_mangle]
pub extern "C" fn tfhe_compress(
    server_key_bytes: *const u8, server_key_len: usize,
    ct_bytes: *const u8, ct_len: usize,
    out_ct: *mut *mut u8, out_ct_len: *mut usize,
) -> i32 {
    let result = panic::catch_unwind(|| {
        let sk_slice = unsafe { slice::from_raw_parts(server_key_bytes, server_key_len) };
        let ct_slice = unsafe { slice::from_raw_parts(ct_bytes, ct_len) };
        let server_key: tfhe::ServerKey = bincode::deserialize(sk_slice).map_err(|_| ())?;
        set_server_key(server_key);
        let ct: FheUint32 = bincode::deserialize(ct_slice).map_err(|_| ())?;
        let compressed = ct.compress();
        let bytes = bincode::serialize(&compressed).map_err(|_| ())?;
        Ok::<_, ()>(bytes)
    });
    match result {
        Ok(Ok(bytes)) => { write_bytes_out(bytes, out_ct, out_ct_len); 0 }
        _ => -1,
    }
}

#[no_mangle]
pub extern "C" fn tfhe_decompress(
    server_key_bytes: *const u8, server_key_len: usize,
    compressed_bytes: *const u8, compressed_len: usize,
    out_ct: *mut *mut u8, out_ct_len: *mut usize,
) -> i32 {
    let result = panic::catch_unwind(|| {
        let sk_slice = unsafe { slice::from_raw_parts(server_key_bytes, server_key_len) };
        let c_slice = unsafe { slice::from_raw_parts(compressed_bytes, compressed_len) };
        let server_key: tfhe::ServerKey = bincode::deserialize(sk_slice).map_err(|_| ())?;
        set_server_key(server_key);
        let compressed: tfhe::CompressedFheUint32 =
            bincode::deserialize(c_slice).map_err(|_| ())?;
        let ct: FheUint32 = compressed.decompress();
        let bytes = bincode::serialize(&ct).map_err(|_| ())?;
        Ok::<_, ()>(bytes)
    });
    match result {
        Ok(Ok(bytes)) => { write_bytes_out(bytes, out_ct, out_ct_len); 0 }
        _ => -1,
    }
}

#[no_mangle]
pub extern "C" fn tfhe_free_bytes(ptr: *mut u8, len: usize) {
    if ptr.is_null() || len == 0 { return; }
    unsafe { let _ = Vec::from_raw_parts(ptr, len, len); }
}