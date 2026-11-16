import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../state/account_model.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _email = TextEditingController();
  final _password = TextEditingController();
  String? _error;
  bool _busy = false;

  Future<void> _submit() async {
    setState(() => _busy = true);
    try {
      await context.read<AccountModel>().login(_email.text.trim(), _password.text);
      if (mounted) context.go('/');
    } catch (e) {
      setState(() => _error = e is String ? e : 'Không đăng nhập được, thử lại sau');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  void dispose() {
    _email.dispose();
    _password.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
        body: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              TextField(
                controller: _email,
                keyboardType: TextInputType.emailAddress,
                autofillHints: const [AutofillHints.email],
                decoration: const InputDecoration(labelText: 'Email'),
              ),
              TextField(
                controller: _password,
                obscureText: true,
                autofillHints: const [AutofillHints.password],
                decoration: const InputDecoration(labelText: 'Mật khẩu'),
              ),
              if (_error != null) Text(_error!, style: const TextStyle(color: Colors.red)),
              FilledButton(
                onPressed: _busy ? null : _submit,
                child: const Text('Đăng nhập'),
              ),
            ],
          ),
        ),
      );
}
